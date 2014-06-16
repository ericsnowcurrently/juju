// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package restore

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"labix.org/v2/mgo"
	"launchpad.net/goyaml"

	"github.com/juju/juju/environs"
	"github.com/juju/juju/juju"
	"github.com/juju/juju/state"
	"github.com/juju/juju/worker/peergrouper"
)

// TODO Make this, tar and untar into a b&r utils pkg
var runCommand = _runCommand

func _runCommand(cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	out, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	if _, ok := err.(*exec.ExitError); ok && len(out) > 0 {
		return fmt.Errorf("error executing %q: %s", cmd, strings.Replace(string(out), "\n", "; ", -1))
	}
	return fmt.Errorf("cannot execute %q: %v", cmd, err)
}

func untarFiles(tarFile, outputFolder string, compressed bool) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return fmt.Errorf("cannot open backup file %q: %v", tarFile, err)
	}
	defer f.Close()
	var r io.Reader = f
	if compressed {
		r, err = gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("cannot uncompress tar file %q: %v", tarFile, err)
		}
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return fmt.Errorf("failed while reading tar header: %v", err)
		}
		buf := make([]byte, hdr.Size)
		buf, err = ioutil.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("failed while reading tar contents: %v", err)
		}
		fullPath := filepath.Join(outputFolder, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			if err = os.MkdirAll(fullPath, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("cannot extract directory %q: %v", fullPath, err)
			}
		} else {
			fh, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("some of the tar contents cannot be written to disk: %v", err)
			}
			_, err = fh.Write(buf)

			if err != nil {
				fh.Close()
				return fmt.Errorf("some of the tar contents cannot be written to disk: %v", err)
			}
			err = fh.Chmod(os.FileMode(hdr.Mode))
			fh.Close()
			if err != nil {
				return fmt.Errorf("cannot set proper mode on file %q: %v", fullPath, err)
			}

		}
	}
	return nil
}

func updateStateServersRecords() error {
	return nil
}

// resetReplicaSet re-initiates replica-set using the new state server
// values, this is required after a mongo restore.
// in case of failure returns error
func resetReplicaSet(dialInfo *mgo.DialInfo, privateAddress string,
	agentConf agentConfig) error {
	user := "admin"
	password := agentConf.Credentials.AdminPassword
	memberHostPort := fmt.Sprintf("%s:%s", privateAddress, agentConf.ApiPort)

	params := peergrouper.InitiateMongoParams{dialInfo,
		memberHostPort,
		user,
		password,
	}
	return peergrouper.InitiateMongoServer(params, true)
}

var getReplaceableFiles = _getReplaceableFiles

func _getReplaceableFiles() []string {
	return []string{
		"/var/lib/juju/db",
		"/var/lib/juju",
		"/var/log/juju",
	}
}

var getFilesystemRoot = _getFilesystemRoot

func _getFilesystemRoot() string {
	return string(os.PathSeparator)
}

// prepareMachineForBackup deletes all files from the re-bootstrapped
// machine that are to be replaced by the backup and recreates those
// directories that are to contain new files; this is to avoid
// possible mixup from new/old files that lead to an inconsistent
// restored state machine.
func prepareMachineForBackup() error {
	replaceableFiles := getReplaceableFiles()
	for _, toBeRecreated := range replaceableFiles {
		//XXX (hduran-8) this can be improved by storing the mode and
		// sorting the paths before creating the dirs
		originalDirInfo, err := os.Stat(toBeRecreated)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(toBeRecreated); err != nil {
			return err
		}
		if err := os.MkdirAll(toBeRecreated, originalDirInfo.Mode()); err != nil {
			return err
		}
	}
	return nil
}

// newDialInfo returns mgo.DialInfo with the given address using the minimal
// possible setup
func newDialInfo(privateAddr string, conf agentConfig) mgo.DialInfo {
	return mgo.DialInfo{
		Addrs:    []string{fmt.Sprintf("%s:%s", privateAddr, conf.ApiPort)},
		Timeout:  30 * time.Second,
		Username: conf.Credentials.AdminUsername,
		Password: conf.Credentials.AdminPassword,
	}
}

var getMongorestorePath = _getMongorestorePath

func _getMongorestorePath() (string, error) {
	const mongoDumpPath string = "/usr/lib/juju/bin/mongorestore"

	if _, err := os.Stat(mongoDumpPath); err == nil {
		return mongoDumpPath, nil
	}

	path, err := exec.LookPath("mongorestore")
	if err != nil {
		return "", err
	}
	return path, nil
}

var getMongoDbPath = _getMongoDbPath

func _getMongoDbPath() string {
	return "/var/lib/juju/db"
}

// placeNewMongo tries to use mongorestore to replace an existing
// mongo (obtained from getMongoDbPath) with the dump in newMongoDumpPath
// returns an error if its not possible
func placeNewMongo(newMongoDumpPath string) error {
	mongoRestore, err := getMongorestorePath()
	if err != nil {
		return fmt.Errorf("mongorestore not available: %v", err)
	}
	err = runCommand(
		mongoRestore,
		"--drop",
		"--oplogReplay",
		"--dbpath", getMongoDbPath(),
		newMongoDumpPath)

	if err != nil {
		return fmt.Errorf("failed to restore database dump: %v", err)
	}

	return nil
}

type credentials struct {
	Tag           string
	TagPassword   string
	AdminUsername string
	AdminPassword string
}

type agentConfig struct {
	Credentials credentials
	ApiPort     string
	StatePort   string
}

// fetchAgentConfigFromBackup parses <dataDir>/machine-0/agents/machine-0/agent.conf
// and returns an agentConfig struct filled with the data that will not change
// from the backed up one (tipically everything but the hosts)
func fetchAgentConfigFromBackup(agentConfigFilePath string) (agentConfig, error) {
	agentConf, err := os.Open(agentConfigFilePath)
	if err != nil {
		return agentConfig{}, err
	}
	defer agentConf.Close()

	data, err := ioutil.ReadAll(agentConf)
	if err != nil {
		return agentConfig{}, fmt.Errorf("failed to read agent config file: %v", err)
	}
	var conf interface{}
	if err := goyaml.Unmarshal(data, &conf); err != nil {
		return agentConfig{}, fmt.Errorf("cannot unmarshal agent config file: %v", err)
	}
	m, ok := conf.(map[interface{}]interface{})
	if !ok {
		return agentConfig{}, fmt.Errorf("config file unmarshalled to %T not %T", conf, m)
	}

	tagUser, ok := m["tag"].(string)
	if !ok || tagUser == "" {
		return agentConfig{}, fmt.Errorf("tag not found in configuration")
	}

	tagPassword, ok := m["statepassword"].(string)
	if !ok || tagPassword == "" {
		return agentConfig{}, fmt.Errorf("agent tag user password not found in configuration")
	}

	adminPassword, ok := m["oldpassword"].(string)
	if !ok || adminPassword == "" {
		return agentConfig{}, fmt.Errorf("agent admin password not found in configuration")
	}

	statePortNum, ok := m["stateport"].(int)
	if !ok {
		return agentConfig{}, fmt.Errorf("state port not found in configuration")
	}
	statePort := strconv.Itoa(statePortNum)

	apiPortNum, ok := m["apiport"].(int)
	if !ok {
		return agentConfig{}, fmt.Errorf("api port not found in configuration")
	}
	apiPort := strconv.Itoa(apiPortNum)

	return agentConfig{
		Credentials: credentials{
			Tag:           tagUser,
			TagPassword:   tagPassword,
			AdminUsername: "admin",
			AdminPassword: adminPassword,
		},
		StatePort: statePort,
		ApiPort:   apiPort,
	}, nil
}

// updateAllMachines finds all machines and resets the stored state address
// in each of them. The address does not include the port.
func updateAllMachines(environ environs.Environ, stateAddr string) error {
	conn, err := juju.NewConn(environ)
	if err != nil {
		return err
	}
	st := conn.State
	machines, err := st.AllMachines()
	if err != nil {
		return err
	}
	pendingMachineCount := 0
	done := make(chan error)
	for _, machine := range machines {
		// A newly resumed state server requires no updating, and more
		// than one state server is not yet support by this code.
		if machine.IsManager() || machine.Life() == state.Dead {
			continue
		}
		pendingMachineCount++
		machine := machine
		go func() {
			//err := runMachineUpdate(machine, setAgentAddressScript(stateAddr))
			err := fmt.Errorf("delete me") // TODO: Create an actual update for machines
			if err != nil {
				fmt.Errorf("failed to update machine %s: %v", machine, err)
			}
			done <- err
		}()
	}
	err = nil
	for ; pendingMachineCount > 0; pendingMachineCount-- {
		if updateErr := <-done; updateErr != nil && err == nil {
			err = fmt.Errorf("machine update failed")
		}
	}
	return err
}

func updateStateServerAddress(newAddress string, agentConf agentConfig) {
	// TODO: Use te same function that jujud command to update api/state server
}

// Restore extracts the content of the given backup file and:
// * runs mongorestore with the backed up mongo dump
// * updates and writes configuration files
// * updates existing db entries to make sure they hold no references to
// old instances
func Restore(backupFile, privateAddress string, environ environs.Environ) error {
	workDir := os.TempDir()
	defer os.RemoveAll(workDir)

	// XXX (perrito666) obtain this from the proper place here and in backup
	const agentFile string = "var/lib/juju/agents/machine-0/agent.conf"

	// Extract outer container
	if err := untarFiles(backupFile, workDir, true); err != nil {
		return fmt.Errorf("cannot extract files from backup: %v", err)
	}
	if err := prepareMachineForBackup(); err != nil {
		return fmt.Errorf("cannot delete existing files: %v", err)
	}
	backupFilesPath := filepath.Join(workDir, "juju-backup")
	// Extract inner container
	innerBackup := filepath.Join(backupFilesPath, "root.tar")
	if err := untarFiles(innerBackup, getFilesystemRoot(), false); err != nil {
		return fmt.Errorf("cannot obtain system files from backup: %v", err)
	}
	// Restore backed up mongo
	mongoDump := filepath.Join(backupFilesPath, "juju-backup", "dump")
	if err := placeNewMongo(mongoDump); err != nil {
		return fmt.Errorf("error restoring state from backup: %v", err)
	}
	// Load configuration values that are to remain
	agentConf, err := fetchAgentConfigFromBackup(agentFile)
	if err != nil {
		return fmt.Errorf("cannot obtain agent configuration information: %v", err)
	}
	// Re-start replicaset with the new value for server address
	dialInfo := newDialInfo(privateAddress, agentConf)
	resetReplicaSet(&dialInfo, privateAddress, agentConf)
	updateStateServerAddress(privateAddress, agentConf) // XXX Stub

	privateHostPorts := fmt.Sprintf("%s:%d", privateAddress, agentConf.StatePort)
	err = updateAllMachines(environ, privateHostPorts)
	if err != nil {
		return fmt.Errorf("cannot update agents: %v", err)
	}

	return nil
}
