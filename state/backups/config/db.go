// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juju/errors"

	"github.com/juju/juju/environmentserver/authentication"
	"github.com/juju/juju/mongo"
)

const dbDumpBinary = "mongodump"

var (
	dbStartup = mongo.ServiceName("") + ".conf"
	dbBinDir  = filepath.Dir(mongo.JujuMongodPath)
)

// DBConnInfo is a simplification of authentication.MongoInfo.
type DBConnInfo interface {
	// Address returns the connection address.
	Address() string
	// Username returns the connection username.
	Username() string
	// Password returns the connection password.
	Password() string
}

type dbConnInfo struct {
	address  string
	username string
	password string
}

// NewDBConnInfo returns a new DBConnInfo.
func NewDBConnInfo(addr, user, pw string) DBConnInfo {
	info := dbConnInfo{
		address:  addr,
		username: user,
		password: pw,
	}
	return &info
}

func (ci *dbConnInfo) Address() string {
	return ci.address
}

func (ci *dbConnInfo) Username() string {
	return ci.username
}

func (ci *dbConnInfo) Password() string {
	return ci.password
}

// UpdateFromMongoInfo pulls in the provided connection info.
func (ci *dbConnInfo) UpdateFromMongoInfo(mgoInfo *authentication.MongoInfo) {
	ci.address = mgoInfo.Addrs[0]
	ci.password = mgoInfo.Password

	// TODO(dfc) Backup should take a Tag.
	if mgoInfo.Tag != nil {
		ci.username = mgoInfo.Tag.String()
	}
}

// DBInfo is an abstraction of information for the juju database.
type DBInfo interface {
	// ConnInfo returns the connection-related DB info.
	ConnInfo() DBConnInfo
	// StartupFile returns the name of the DB startup file (e.g. upstart).
	StartupFile() string
	// BinDir() returns the name of the directory of DB executables.
	BinDir() string
	// DumpBinary returns the path to the db dump executable.
	DumpBinary() string
	// DumpArgs returns the list of arguments to the dump command.
	DumpArgs(outDir string) []string
}

type dbInfo struct {
	connInfo DBConnInfo

	startupFile string

	binDir string
}

// NewDBInfo returns a new DBInfo.
func NewDBInfo(startup, binDir string, connInfo DBConnInfo) *dbInfo {
	if startup == "" {
		startup = dbStartup
	}
	if binDir == "" {
		binDir = dbBinDir
	}
	if connInfo == nil {
		connInfo = &dbConnInfo{}
	}

	info := dbInfo{
		connInfo: connInfo,

		startupFile: startup,

		binDir: binDir,
	}
	return &info
}

func (db *dbInfo) ConnInfo() DBConnInfo {
	return db.connInfo
}

func (db *dbInfo) StartupFile() string {
	return db.startupFile
}

func (db *dbInfo) BinDir() string {
	return db.binDir
}

func (db *dbInfo) DumpBinary() string {
	filename, err := find(dbDumpBinary, db.BinDir()+string(os.PathSeparator))
	if err != nil {
		logger.Criticalf(err.Error())
		panic(err)
	}
	return filename
}

func (db *dbInfo) DumpArgs(outDir string) []string {
	return []string{
		"--oplog",
		"--ssl",
		"--host", db.ConnInfo().Address(),
		"--username", db.ConnInfo().Username(),
		"--password", db.ConnInfo().Password(),
		"--out", outDir,
	}
}

func find(name string, initial ...string) (string, error) {
	for _, filename := range initial {
		if strings.HasSuffix(filename, string(os.PathSeparator)) {
			filename = filepath.Join(filename, name)
		}
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	filename, err := exec.LookPath(name)
	if err == nil {
		return filename, nil
	} else if err == exec.ErrNotFound {
		return "", errors.NotFoundf(name)
	} else {
		return "", errors.Trace(err)
	}
}
