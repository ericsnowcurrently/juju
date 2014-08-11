// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package config

import (
	"path/filepath"
)

// Config is an abstraction of the info needed for backups.
type Config interface {
	// CriticalFiles returns the path to each file critical to the
	// operation if juju's state.
	CriticalFiles() []string
}

// SubConfig is an abstraction of the config files for a part of juju.
type SubConfig interface {
	// DataFiles returns the names of files in the state data directory.
	DataFiles() []string
	// StartupFiles returns the names of the startup files.
	StartupFiles() []string
	// LogFiles returns the names of log files.
	LogFiles() []string
	// LoggingConfFiles returns the names of logging config files.
	LoggingConfFiles() []string
	// SSHFiles returns the names of ssh-related files.
	SSHFiles() []string
}

type config struct {
	paths   StatePaths
	configs []SubConfig
}

// NewConfig returns a new backups config.
func NewConfig(paths StatePaths, configs ...SubConfig) Config {
	conf := config{
		paths:   paths,
		configs: configs,
	}
	return &conf
}

func NewDefaultConfig(configs ...SubConfig) Config {
	paths := NewStatePaths("", "", "", "", "")
	conf := NewConfig(
		paths,
		NewToolsConfig(),
		NewMachinesConfig(),
		configs...,
	)
	return conf
}

func (c *config) CriticalFiles() []string {
	files := []string{}

	dataDir := c.paths.DataDir()
	for _, sub := range c.configs {
		for _, name := range sub.DataFiles() {
			files = append(files, filepath.Join(dataDir, name))
		}
	}

	startupDir := c.paths.StartupDir()
	for _, sub := range c.configs {
		for _, name := range sub.StartupFiles() {
			files = append(files, filepath.Join(startupDir, name))
		}
	}

	logsDir := c.paths.LogsDir()
	for _, sub := range c.configs {
		for _, name := range sub.LogFiles() {
			files = append(files, filepath.Join(logsDir, name))
		}
	}

	loggingConfDir := c.paths.LoggingConfDir()
	for _, sub := range c.configs {
		for _, name := range sub.LoggingConfFiles() {
			files = append(files, filepath.Join(loggingConfDir, name))
		}
	}

	sshDir := c.paths.SSHDir()
	for _, sub := range c.configs {
		for _, name := range sub.SSHFiles() {
			files = append(files, filepath.Join(sshDir, name))
		}
	}

	return files
}
