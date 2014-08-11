// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package config

import (
	"github.com/juju/errors"
)

// BackupsConfig is an abstraction of the info needed for backups.
type BackupsConfig interface {
	// FilesToBackUp returns the list of paths to files that should be
	// backed up.
	FilesToBackUp() []string

	// DBDump returns the necessary information to call the dump command.
	DBDump(outDir string) (bin string, args []string, err error)
}

type backupsConfig struct {
	db     DBConfig
	config Config
}

// NewBackupsConfig returns a new backups config.
func NewBackupsConfig(db DBConfig, config Config) BackupsConfig {
	if db == nil {
		db = NewDBConfig()
	}
	if config == nil {
		config = NewDefaultConfig(db)
	}

	conf := backupsConfig{
		db:     db,
		config: config,
	}
	return &conf
}

func (bc *backupsConfig) FilesToBackUp() []string {
	return bc.config.CriticalFiles()
}

func (bc *backupsConfig) DBDump(outDir string) (string, []string, error) {
	bin, args, err := bc.db.Dump(outDir)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	return bin, args, nil
}
