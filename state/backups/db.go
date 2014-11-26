// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/utils/set"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/mongo"
	"github.com/juju/juju/state"
	"github.com/juju/juju/utils"
	"github.com/juju/juju/version"
)

// db is a surrogate for the proverbial DB layer abstraction that we
// wish we had for juju state.  To that end, the package holds the DB
// implementation-specific details and functionality needed for backups.
// Currently that means mongo-specific details.  However, as a stand-in
// for a future DB layer abstraction, the db package does not expose any
// low-level details publicly.  Thus the backups implementation remains
// oblivious to the underlying DB implementation.

var runCommand = utils.RunCommand

// DBConnInfo is a simplification of authentication.MongoInfo, focused
// on the needs of juju state backups.  To ensure that the info is valid
// for use in backups, use the Check() method to get the contained
// values.
type DBConnInfo struct {
	// Address is the DB system's host address.
	Address string
	// Username is used when connecting to the DB system.
	Username string
	// Password is used when connecting to the DB system.
	Password string
}

// DBInfo wraps all the DB-specific information backups needs to dump
// and restore the database.
type DBInfo struct {
	DBConnInfo
	// Targets is a list of databases to dump.
	Targets set.Strings
}

// ignoredDatabases is the list of databases that should not be
// backed up.
var ignoredDatabases = set.NewStrings(
	"backups",
	"presence",
)

// NewDBBackupInfo returns the information needed by backups to dump
// the database.
func NewDBBackupInfo(st *state.State) (*DBInfo, error) {
	connInfo := newMongoConnInfo(st.MongoConnectionInfo())
	targets, err := getBackupTargetDatabases(st)
	if err != nil {
		return nil, errors.Trace(err)
	}

	info := DBInfo{
		DBConnInfo: *connInfo,
		Targets:    targets,
	}
	return &info, nil
}

func newMongoConnInfo(mgoInfo *mongo.MongoInfo) *DBConnInfo {
	info := DBConnInfo{
		Address:  mgoInfo.Addrs[0],
		Password: mgoInfo.Password,
	}

	// TODO(dfc) Backup should take a Tag.
	if mgoInfo.Tag != nil {
		info.Username = mgoInfo.Tag.String()
	}

	return &info
}

func getBackupTargetDatabases(st *state.State) (set.Strings, error) {
	dbNames, err := st.MongoSession().DatabaseNames()
	if err != nil {
		return nil, errors.Annotate(err, "unable to get DB names")
	}

	targets := set.NewStrings(dbNames...).Difference(ignoredDatabases)
	return targets, nil
}

const dumpName = "mongodump"

// DBDumper is any type that dumps something to a dump dir.
type DBDumper interface {
	// Dump something to dumpDir.
	Dump(dumpDir string) error
}

var getMongodumpPath = func() (string, error) {
	mongod, err := mongo.Path()
	if err != nil {
		return "", errors.Annotate(err, "failed to get mongod path")
	}
	mongoDumpPath := filepath.Join(filepath.Dir(mongod), dumpName)

	if _, err := os.Stat(mongoDumpPath); err == nil {
		// It already exists so no need to continue.
		return mongoDumpPath, nil
	}

	path, err := exec.LookPath(dumpName)
	if err != nil {
		return "", errors.Trace(err)
	}
	return path, nil
}

type mongoDumper struct {
	*DBInfo
	// binPath is the path to the dump executable.
	binPath string
}

// NewDBDumper returns a new value with a Dump method for dumping the
// juju state database.
func NewDBDumper(info *DBInfo) (DBDumper, error) {
	mongodumpPath, err := getMongodumpPath()
	if err != nil {
		return nil, errors.Annotate(err, "mongodump not available")
	}

	dumper := mongoDumper{
		DBInfo:  info,
		binPath: mongodumpPath,
	}
	return &dumper, nil
}

func (md *mongoDumper) options(dumpDir string) []string {
	options := []string{
		"--ssl",
		"--authenticationDatabase", "admin",
		"--host", md.Address,
		"--username", md.Username,
		"--password", md.Password,
		"--out", dumpDir,
		"--oplog",
	}
	return options
}

func (md *mongoDumper) dump(dumpDir string) error {
	options := md.options(dumpDir)
	if err := runCommand(md.binPath, options...); err != nil {
		return errors.Annotate(err, "error dumping databases")
	}
	return nil
}

// Dump dumps the juju state-related databases.  To do this we dump all
// databases and then remove any ignored databases from the dump results.
func (md *mongoDumper) Dump(baseDumpDir string) error {
	if err := md.dump(baseDumpDir); err != nil {
		return errors.Trace(err)
	}

	found, err := listDatabases(baseDumpDir)
	if err != nil {
		return errors.Trace(err)
	}

	// Strip the ignored database from the dump dir.
	ignored := found.Difference(md.Targets)
	err = stripIgnored(ignored, baseDumpDir)
	return errors.Trace(err)
}

// stripIgnored removes the ignored DBs from the mongo dump files.
// This involves deleting DB-specific directories.
func stripIgnored(ignored set.Strings, dumpDir string) error {
	for _, dbName := range ignored.Values() {
		if dbName != "backups" {
			// We allow all ignored databases except "backups" to be
			// included in the archive file.  Restore will be
			// responsible for deleting those databases after
			// restoring them.
			continue
		}
		dirname := filepath.Join(dumpDir, dbName)
		if err := os.RemoveAll(dirname); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// listDatabases returns the name of each sub-directory of the dump
// directory.  Each corresponds to a database dump generated by
// mongodump.  Note that, while mongodump is unlikely to change behavior
// in this regard, this is not a documented guaranteed behavior.
func listDatabases(dumpDir string) (set.Strings, error) {
	list, err := ioutil.ReadDir(dumpDir)
	if err != nil {
		return set.Strings{}, errors.Trace(err)
	}

	databases := make(set.Strings)
	for _, info := range list {
		if !info.IsDir() {
			// Notably, oplog.bson is thus excluded here.
			continue
		}
		databases.Add(info.Name())
	}
	return databases, nil
}

// runExternalCommand it is intended to be a wrapper around utils.RunCommand
// its main objective is to provide a suiteable output for our use case.
func runExternalCommand(cmd string, args ...string) error {
	if out, err := utils.RunCommand(cmd, args...); err != nil {
		if _, ok := err.(*exec.ExitError); ok && len(out) > 0 {
			return errors.Annotatef(err, "error executing %q: %s", cmd, strings.Replace(string(out), "\n", "; ", -1))
		}
		return errors.Annotatef(err, "cannot execute %q", cmd)
	}
	return nil
}

// mongorestorePath will look for mongorestore binary on the system
// and return it if mongorestore actually exists.
// it will look first for the juju provided one and if not found make a
// try at a system one.
func mongorestorePath() (string, error) {
	const mongoRestoreFullPath string = "/usr/lib/juju/bin/mongorestore"

	if _, err := os.Stat(mongoRestoreFullPath); err == nil {
		return mongoRestoreFullPath, nil
	}

	path, err := exec.LookPath("mongorestore")
	if err != nil {
		return "", errors.Trace(err)
	}
	return path, nil
}

// mongoRestoreArgsForVersion returns a string slice containing the args to be used
// to call mongo restore since these can change depending on the backup method.
// Version 0: a dump made with --db, stoping the state server.
// Version 1: a dump made with --oplog with a running state server.
// TODO (perrito666) change versions to use metadata version
func mongoRestoreArgsForVersion(ver version.Number, dumpPath string) ([]string, error) {
	dbDir := filepath.Join(agent.DefaultDataDir, "db")
	switch {
	case ver.Major == 1 && ver.Minor < 22:
		return []string{"--drop", "--dbpath", dbDir, dumpPath}, nil
	case ver.Major == 1 && ver.Minor >= 22:
		return []string{"--drop", "--oplogReplay", "--dbpath", dbDir, dumpPath}, nil
	default:
		return nil, errors.Errorf("this backup file is incompatible with the current version of juju")
	}
}

// PlaceNewMongo tries to use mongorestore to replace an existing
// mongo with the dump in newMongoDumpPath returns an error if its not possible.
func PlaceNewMongo(newMongoDumpPath string, ver version.Number) error {
	mongoRestore, err := mongorestorePath()
	if err != nil {
		return errors.Annotate(err, "mongorestore not available")
	}

	mgoRestoreArgs, err := mongoRestoreArgsForVersion(ver, newMongoDumpPath)
	if err != nil {
		return errors.Errorf("cannot restore this backup version")
	}
	err = runExternalCommand("initctl", "stop", "juju-db")
	if err != nil {
		return errors.Annotate(err, "failed to stop mongo")
	}

	err = runExternalCommand(mongoRestore, mgoRestoreArgs...)

	if err != nil {
		return errors.Annotate(err, "failed to restore database dump")
	}

	err = runExternalCommand("initctl", "start", "juju-db")
	if err != nil {
		return errors.Annotate(err, "failed to start mongo")
	}

	return nil
}
