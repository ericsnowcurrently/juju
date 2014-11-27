// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build !windows !linux

package restore

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/juju/errors"
	"github.com/juju/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/mongo"
	"github.com/juju/juju/network"
	"github.com/juju/juju/state"
	"github.com/juju/juju/utils/ssh"
	"github.com/juju/juju/worker/peergrouper"
)

var filesystemRoot = getFilesystemRoot

func getFilesystemRoot() string {
	return "/"
}

// newDialInfo returns mgo.DialInfo with the given address using the minimal
// possible setup.
func newDialInfo(privateAddr string, conf agent.Config) (*mgo.DialInfo, error) {
	dialOpts := mongo.DialOpts{Direct: true}
	ssi, ok := conf.StateServingInfo()
	if !ok {
		errors.Errorf("cannot get state serving info to dial")
	}
	info := mongo.Info{
		Addrs:  []string{fmt.Sprintf("%s:%d", privateAddr, ssi.StatePort)},
		CACert: conf.CACert(),
	}
	dialInfo, err := mongo.DialInfo(info, dialOpts)
	if err != nil {
		return nil, errors.Annotate(err, "cannot produce a dial info")
	}
	dialInfo.Username = "admin"
	dialInfo.Password = conf.OldPassword()
	return dialInfo, nil
}

// updateMongoEntries will update the machine entries in the restored mongo to
// reflect the real machine instanceid in case it changed (a newly bootstraped
// server).
func updateMongoEntries(newInstId instance.Id, st *state.State) error {
	session, err := st.MongoSession.Copy()
	if err != nil {
		return errors.Annotate(err, "cannot connect to mongo to update")
	}
	defer session.Close()

	coll := session.DB("juju").C("machines")
	err = coll.Update(
		bson.M{"machineid": "0"},
		bson.M{"$set": bson.M{"instanceid": string(newInstId)}},
	)
	if err != nil {
		return errors.Annotate(err, "cannot update machine 0 instance information")
	}
	return nil
}

// newStateConnection tries to connect to the newly restored state server.
func newStateConnection(agentConf agent.Config) (*state.State, error) {
	// We need to retry here to allow mongo to come up on the restored state server.
	// The connection might succeed due to the mongo dial retries but there may still
	// be a problem issuing database commands.
	var (
		st  *state.State
		err error
	)
	info, ok := agentConf.MongoInfo()
	if !ok {
		return nil, errors.Errorf("cannot retrieve info to connect to mongo")
	}
	attempt := utils.AttemptStrategy{Delay: 15 * time.Second, Min: 8}
	for a := attempt.Start(); a.Next(); {
		st, err = state.Open(info, mongo.DefaultDialOpts(), environs.NewStatePolicy())
		if err == nil {
			break
		}
		logger.Errorf("cannot open state, retrying: %v", err)
	}
	return st, errors.Annotate(err, "cannot open state")
}

// updateAllMachines finds all machines and resets the stored state address
// in each of them. The address does not include the port.
func updateAllMachines(privateAddress string, st *state.State) error {
	machines, err := st.AllMachines()
	if err != nil {
		return errors.Trace(err)
	}
	pendingMachineCount := 0
	done := make(chan error)
	for key := range machines {
		// key is used to have machine be scope bound to the loop iteration.
		machine := machines[key]
		// A newly resumed state server requires no updating, and more
		// than one state server is not yet supported by this code.
		if machine.IsManager() || machine.Life() == state.Dead {
			continue
		}
		pendingMachineCount++
		go func() {
			err := runMachineUpdate(machine, setAgentAddressScript(privateAddress))
			done <- errors.Annotatef(err, "failed to update machine %s", machine)
		}()
	}
	for ; pendingMachineCount > 0; pendingMachineCount-- {
		if updateErr := <-done; updateErr != nil && err == nil {
			logger.Errorf("failed updating machine: %v", err)
		}
	}
	// We should return errors encapsulated in a digest.
	return nil
}

// agentAddressTemplate is the template used to replace the api server data
// in the agents for the new ones if the machine has been rebootstraped.
var agentAddressTemplate = template.Must(template.New("").Parse(`
set -xu
cd /var/lib/juju/agents
for agent in *
do
	status  jujud-$agent| grep -q "^jujud-$agent start" > /dev/null
	if [ $? -eq 0 ]; then
		initctl stop jujud-$agent 
	fi
	sed -i.old -r "/^(stateaddresses|apiaddresses):/{
		n
		s/- .*(:[0-9]+)/- {{.Address}}\1/
	}" $agent/agent.conf

	# If we're processing a unit agent's directly
	# and it has some relations, reset
	# the stored version of all of them to
	# ensure that any relation hooks will
	# fire.
	if [[ $agent = unit-* ]]
	then
		find $agent/state/relations -type f -exec sed -i -r 's/change-version: [0-9]+$/change-version: 0/' {} \;
	fi
	# Just in case is a stale unit
	status  jujud-$agent| grep -q "^jujud-$agent stop" > /dev/null
	if [ $? -eq 0 ]; then
		initctl start jujud-$agent
	fi
done
`))

func execTemplate(tmpl *template.Template, data interface{}) string {
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		panic(errors.Annotate(err, "template error"))
	}
	return buf.String()
}

// setAgentAddressScript generates an ssh script argument to update state addresses.
func setAgentAddressScript(stateAddr string) string {
	return execTemplate(agentAddressTemplate, struct {
		Address string
	}{stateAddr})
}

// runMachineUpdate connects via ssh to the machine and runs the update script.
func runMachineUpdate(m *state.Machine, sshArg string) error {
	addr := network.SelectPublicAddress(m.Addresses())
	if addr == "" {
		return errors.Errorf("no appropriate public address found")
	}
	return runViaSSH(addr, sshArg)
}

// runViaSSH runs script in the remote machine with address addr.
func runViaSSH(addr string, script string) error {
	// This is taken from cmd/juju/ssh.go there is no other clear way to set user
	userAddr := "ubuntu@" + addr
	sshOptions := ssh.Options{}
	sshOptions.SetIdentities("/var/lib/juju/system-identity")
	userCmd := ssh.Command(userAddr, []string{"sudo", "-n", "bash", "-c " + utils.ShQuote(script)}, &sshOptions)
	var stderrBuf bytes.Buffer
	userCmd.Stderr = &stderrBuf
	err := userCmd.Run()
	if err != nil {
		return errors.Annotatef(err, "ssh command failed: %q", stderrBuf.String())
	}
	return nil
}

// backupVersion will use information from the backup file and metadata (if available)
// to determine which backup version this file belongs to.
// Once Metadata is given a version option we can version backups
// we could use juju version to signal this, but currently:
// Version 0: juju backup plugin (a bash script)
// Version 1: juju backups create (first implementation) for the
// moment this version is determined by checking for metadata but not
// its contents.
func backupVersion(backupMetadata *Metadata, backupFilesPath string) (int, error) {
	backupMetadataFile := true
	if _, err := os.Stat(filepath.Join(backupFilesPath, "metadata.json")); os.IsNotExist(err) {
		backupMetadataFile = false
	} else if err != nil {
		return 0, errors.Annotate(err, "cannot read metadata file")
	}
	if backupMetadata == nil && !backupMetadataFile {
		return 0, nil
	}
	return 1, nil
}

//-----------------------------------------

type restorer struct {
	privateAddress string
	newInstId      string
}

func NewRestorer(privateAddress string, newInstId instance.Id) backups.Restorer {
	return &restorer{privateAddress, newInstId}
}

// Restore handles either returning or creating a state server to a backed up status:
// * extracts the content of the given backup file and:
// * runs mongorestore with the backed up mongo dump
// * updates and writes configuration files
// * updates existing db entries to make sure they hold no references to
// old instances
// * updates config in all agents.
func (r *restorer) Restore(archive io.Reader) error {
	workspace, err := backups.NewArchiveWorkspaceReader(archive)
	if err != nil {
		return errors.Annotate(err, "cannot unpack backup file")
	}
	defer workspace.Close()

	// Restore files/data from the backup archive.
	if err := restoreFiles(workspace); err != nil {
		return errors.Trace(err)
	}
	if err := restoreDB(workspace); err != nil {
		return errors.Trace(err)
	}

	// Point to the new state server.
	agentConfig, err := r.fixConfig()
	if err != nil {
		return errors.Trace(err)
	}
	if err := resetReplicaSet(agentConfig); err != nil {
		return errors.Trace(err)
	}
	if err := resetRunningAgents(agentConfig); err != nil {
		return errors.Trace(err)
	}

	// Finalize the restore.
	rInfo, err := st.EnsureRestoreInfo()
	if err != nil {
		return errors.Trace(err)
	}
	// Mark restoreInfo as Finished so upon restart of the apiserver
	// the client can reconnect and determine if we where succesful.
	if err := rInfo.SetStatus(state.RestoreFinished); err != nil {
		return errors.Annotate(err, "failed to set status to finished")
	}

	return nil
}

// resetReplicaSet re-initiates replica-set using the new state server
// values, this is required after a mongo restore.
// In case of failure returns error.
func (r *restorer) resetReplicaSet(agentConfig agent.ConfigSetterWriter) error {
	// Re-start replicaset with the new value for server address
	dialInfo, err := newDialInfo(r.privateAddress, agentConfig)
	if err != nil {
		return errors.Annotate(err, "cannot produce dial information")
	}

	memberHostPort := fmt.Sprintf("%s:%d", privateAddress, ssi.StatePort)
	err = resetReplicaSet(dialInfo, memberHostPort)
	if err != nil {
		return errors.Annotate(err, "cannot reset replicaSet")
	}

	params := peergrouper.InitiateMongoParams{dialInfo,
		memberHostPort,
		dialInfo.Username,
		dialInfo.Password,
	}
	return peergrouper.InitiateMongoServer(params, true)
}

func (r *restorer) resetRunningAgents(agentConfig agent.ConfigSetterWriter) error {
	// From here we work with the restored state server.
	st, err := newStateConnection(agentConfig)
	if err != nil {
		return errors.Trace(err)
	}
	defer st.Close()

	// Update entries for machine 0 to point to the newest instance
	err = updateMongoEntries(r.newInstId, st)
	if err != nil {
		return errors.Annotate(err, "cannot update mongo entries")
	}

	// update all agents known to the new state server.
	// TODO(perrito666): We should never stop process because of this
	// it is too late to go back and errors in a couple of agents have
	// better change of being fixed by the user, if we where to fail
	// we risk an inconsistent state server because of one unresponsive
	// agent, we should nevertheless return the err info to the user.
	// for this updateAllMachines will not return errors for individual
	// agent update failures
	err = updateAllMachines(privateAddress, st)
	if err != nil {
		return errors.Annotate(err, "cannot update agents")
	}

	return nil
}

func restoreFiles(workspace backups.ArchiveWorkspace) error {
	// delete all the files to be replaced
	if err := PrepareMachineForRestore(); err != nil {
		return errors.Annotate(err, "cannot delete existing files")
	}

	if err := workspace.UnpackFilesBundle(filesystemRoot()); err != nil {
		return errors.Annotate(err, "cannot obtain system files from backup")
	}
	return nil
}

func restoreDB(workspace backups.ArchiveWorkspace) error {
	meta, err := workspace.Metadata()
	if err != nil {
		return errors.Annotatef(err, "cannot read metadata file, this backup is either too old or corrupt")
	}
	version := meta.Origin.Version

	// Restore backed up mongo
	if err := PlaceNewMongo(workspace.DBDumpDir, version); err != nil {
		return errors.Annotate(err, "error restoring state from backup")
	}

	return nil
}

func (r *restorer) fixConfig() (agent.ConfigSetterWriter, error) {
	var agentConfig agent.ConfigSetterWriter

	const confFilename = "/var/lib/juju/agents/machine-0/agent.conf"
	if agentConfig, err = agent.ReadConfig(confFilename); err != nil {
		return nil, errors.Annotate(err, "cannot load agent config from disk")
	}

	ssi, ok := agentConfig.StateServingInfo()
	if !ok {
		return nil, errors.Errorf("cannot determine state serving info")
	}

	APIHostPort := network.HostPort{
		Address: network.Address{
			Value: privateAddress,
			Type:  network.DeriveAddressType(privateAddress),
		},
		Port: ssi.APIPort,
	}
	agentConfig.SetAPIHostPorts([][]network.HostPort{{APIHostPort}})
	if err := agentConfig.Write(); err != nil {
		return nil, errors.Annotate(err, "cannot write new agent configuration")
	}

	return agentConfig, nil
}
