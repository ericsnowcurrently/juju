// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// config contains the configuration information for backups.
package config

import (
	"path/filepath"

	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("juju.state.backups")

// Config is an abstraction of the information backups needs to operate.
type Config interface {
	// DBInfo returns the config's DB info.
	DBInfo() DBInfo
	// Files returns the config's file info.
	Files() Files
	// FilesToBackUp returns the names of the files that should be archived.
	FilesToBackUp() []string
	// Machine returns the ID of the machine running the backups.
	Machine() string
}

type config struct {
	dbInfo DBInfo
	files  Files

	machine string
}

// NewConfig returns a new Config.
func NewConfig(dbInfo DBInfo, files Files, machine string) Config {
	conf := config{
		dbInfo: dbInfo,
		files:  files,

		machine: machine,
	}
	return &conf
}

func (c *config) DBInfo() DBInfo {
	return c.dbInfo
}

func (c *config) Files() Files {
	return c.files
}

func (c *config) Machine() string {
	return c.machine
}

func (c *config) FilesToBackUp() []string {
	files := c.Files().CriticalFiles()

	files = append(files, filepath.Join(
		c.Files().StartupDir(),
		c.DBInfo().StartupFile(),
	))

	return files
}
