// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package config

import (
	"path/filepath"

	"github.com/juju/errors"
)

const (
	defaultDataDir        = "/var/lib/juju"
	defaultStartupDir     = "/etc/init"
	defaultLoggingConfDir = "/etc/rsyslog.d"
	defaultLogsDir        = "/var/log/juju"
	defaultSSHDir         = "/home/ubuntu/.ssh"
)

func globFiles(glob string) ([]string, error) {
	files, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotatef(err, "error getting config files (%s)", glob)
	}
	return files, nil
}

func mustGlobFiles(glob string) []string {
	files, err := globFiles(glob)
	if err != nil {
		logger.Criticalf(err.Error())
		panic(err)
	}
	return files
}

// Files is an abstraction of the files important to backups.
type Files interface {
	//
	DataDir() string
	//
	AgentsDir() string
	//
	ToolsDir() string
	//
	StartupDir() string
	//
	LoggingConfDir() string
	//
	LogsDir() string
	//
	SSHDir() string

	//
	AgentsConfFiles() []string
	//
	StartupFiles() []string
	//
	LoggingConfFiles() []string
	//
	SecureFiles() []string
	//
	CriticalLogFiles() []string

	// CriticalFiles returns the list of files critical to juju's operation.
	CriticalFiles() []string
}

type files struct {
	dataDir        string
	startupDir     string
	loggingConfDir string
	logsDir        string
	sshDir         string
}

// NewFiles eturns a new Files value.
func NewFiles(
	dataDir, startupDir string, loggingConfDir, logsDir, sshDir string,
) Files {
	if dataDir == "" {
		dataDir = defaultDataDir
	}
	if startupDir == "" {
		startupDir = defaultStartupDir
	}
	if loggingConfDir == "" {
		loggingConfDir = defaultLoggingConfDir
	}
	if logsDir == "" {
		logsDir = defaultLogsDir
	}
	if sshDir == "" {
		sshDir = defaultSSHDir
	}

	f := files{
		dataDir:        dataDir,
		startupDir:     startupDir,
		loggingConfDir: loggingConfDir,
		logsDir:        logsDir,
		sshDir:         sshDir,
	}
	return &f
}

func (f *files) DataDir() string {
	return f.dataDir
}

func (f *files) StartupDir() string {
	return f.startupDir
}

func (f *files) LoggingConfDir() string {
	return f.loggingConfDir
}

func (f *files) LogsDir() string {
	return f.logsDir
}

func (f *files) SSHDir() string {
	return f.sshDir
}

func (f *files) AgentsDir() string {
	return filepath.Join(f.DataDir(), "agents")
}

func (f *files) ToolsDir() string {
	return filepath.Join(f.DataDir(), "tools")
}

func (f *files) AgentsConfFiles() []string {
	glob := filepath.Join(f.AgentsDir(), "machine-*")
	return mustGlobFiles(glob)
}

func (f *files) StartupFiles() []string {
	glob := filepath.Join(f.StartupDir(), "jujud-machine-*.conf")
	return mustGlobFiles(glob)
}

func (f *files) LoggingConfFiles() []string {
	glob := filepath.Join(f.LoggingConfDir(), "*juju.conf")
	return mustGlobFiles(glob)
}

func (f *files) SecureFiles() []string {
	return []string{
		filepath.Join(f.DataDir(), "server.pem"),
		filepath.Join(f.DataDir(), "system-identity"),
		filepath.Join(f.DataDir(), "nonce.txt"),
		filepath.Join(f.DataDir(), "shared-secret"),

		filepath.Join(f.SSHDir(), "authorized_keys"),
	}
}

func (f *files) CriticalLogFiles() []string {
	return []string{
		filepath.Join(f.LogsDir(), "all-machines.log"),
		filepath.Join(f.LogsDir(), "machine-0.log"),
	}
}

func (f *files) CriticalFiles() []string {
	initConfs := f.StartupFiles()
	agentConfs := f.AgentsConfFiles()
	jujuLogConfs := f.LoggingConfFiles()
	secureFiles := f.SecureFiles()
	logFiles := f.CriticalLogFiles()

	files := []string{
		f.ToolsDir(),
	}
	files = append(files, initConfs...)
	files = append(files, agentConfs...)
	files = append(files, jujuLogConfs...)
	files = append(files, secureFiles...)
	files = append(files, logFiles...)
	return files
}
