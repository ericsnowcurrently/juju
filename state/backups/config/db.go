// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package config

import (
	"os"
	"path/filepath"

	"github.com/juju/errors"

	"github.com/juju/juju/mongo"
)

// TODO(ericsnow) Pull these from elsewhere in juju.
const (
	dbDumpName = "mongodump"
	dbStartup  = "juju-db.conf"
)

// DBConfig is the DB-portion of the backups config.
type DBConfig interface {
	SubConfig
	// ConnInfo returns the config's DB connection info.
	ConnInfo() DBConnInfo
	// Dump returns the necessary information to call the dump command.
	Dump(outDir string) (bin string, args []string, err error)
}

type dbConfig struct {
	connInfo DBConnInfo

	binDir string
}

// NewDBConfig returns a new DBConfig.
func NewDBConfig(connInfo DBConnInfo, binDir string) (DBConfig, error) {
	if bindir == "" {
		// Find the bin dir.
		mongod, err := mongo.Path()
		if err != nil {
			return nil, errors.Annotate(err, "failed to get mongod path")
		}
		binDir = filepath.Dir(mongod)
	}

	conf := dbConfig{
		connInfo: connInfo,
		binDir:   binDir,
	}
	return &conf
}

func (db *dbConfig) StartupFiles() []string {
	return []string{dbStartup}
}

func (db *dbConfig) DataFiles() []string {
	return nil
}

func (db *dbConfig) Dump(outDir string) (bin string, args []string, err error) {
	bin = filepath.Join(db.binDir, dbDumpName)
	if _, err := os.Stat(bin); err != nil {
		return "", nil, errors.Errorf("%q not found: %v", bin, err)
	}

	addr, user, pw, err := db.connInfo.Checked()
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	args = []string{
		"--oplog",
		"--ssl",
		"--host", addr,
		"--username", user,
		"--password", pw,
		"--out", outDir,
	}

	return bin, args, nil
}
