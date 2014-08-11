// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/utils/tar"

	"github.com/juju/juju/environmentserver/authentication"
	"github.com/juju/juju/mongo"
)

// TODO(ericsnow) Pull these from elsewhere in juju.
var (
	defaultDataDir        = "/var/lib/juju"
	defaultSSHDir         = "/home/ubuntu/.ssh"
	defaultStartupDir     = "/etc/init"
	defaultLoggingConfDir = "/etc/rsyslog.d"
	defaultLogsDir        = "/var/log"

	toolsDir     = "tools"
	agentsDir    = "agents"
	machineInits = "jujud-machine-*.conf"
	agentConfs   = "machine-*"
	loggingConfs = "*juju.conf"
	keys         = "authorized_keys"
	allLogs      = "all-machines.log"
	machineLogs  = "machine-0.log"
	secureFiles  = []string{
		"server.pem",
		"system-identity",
		"nonce.txt",
		"shared-secret",
	}
)

var sep = string(os.PathSeparator)

// StateConfig is an abstraction of FS paths related to juju's state.
type StatePaths interface {
	// DataDir returns the state data dir.
	DataDir() string
	// StartupDir returns the system startup dir.
	StartupDir() string
	// LogsDir returns the system logs dir.
	LogsDir() string
	// LoggingConfDir returns the conf dir for the logging system.
	LoggingConfDir() string
	// SSHDir returns the data dir used by juju for SSH-related files.
	SSHDir() string
}

type paths struct {
	dataDir        string
	startupDir     string
	loggingConfDir string
	logsDir        string
	sshDir         string
}

func NewFilesConfig(
	dataDir, startupDir, loggingConfDir, logsDir, sshDir string,
) FilesConfig {
	if dataDir == "" {
		dataDir = defaultDataDir
	}
	if startupDir == "" {
		startupDir = defaultStartupDir
	}
	if loggingConfDir == "" {
		loggingConfDir == defaultLoggingConfDir
	}
	if logsDir == "" {
		logsDir == defaultLogsDir
	}
	if sshDir == "" {
		sshDir = defaultSSHDir
	}

	conf := filesConfig{
		dataDir: dataDir,
		//        toolsDir: filepath.Join(dataDir, toolsDir),
		//        agentsDir: filepath.Join(dataDir, agentsDir),
		sshDir:         sshDir,
		startupDir:     startupDir,
		loggingConfDir: loggingConfDir,
		logsDir:        logsDir,
	}
	return &conf
}

func (fc *filesConfig) DataDir() string {
	return fc.dataDir
}

func (fc *filesConfig) StartupDir() string {
	return fc.startupDir
}

func (fc *filesConfig) LoggingConfDir() string {
	return fc.loggingConfDir
}

func (fc *filesConfig) LogsDir() string {
	return fc.logsDir
}

func (fc *filesConfig) SSHDir() string {
	return fc.sshDir
}

var getFilesToBackup = func() ([]string, error) {
	const dataDir string = "/var/lib/juju"
	initMachineConfs, err := filepath.Glob("/etc/init/jujud-machine-*.conf")
	if err != nil {
		return nil, errors.Annotate(
			err, "failed to fetch machine upstart files")
	}
	agentConfs, err := filepath.Glob(filepath.Join(
		dataDir, "agents", "machine-*"))
	if err != nil {
		return nil, errors.Annotate(
			err, "failed to fetch agent configuration files")
	}
	jujuLogConfs, err := filepath.Glob("/etc/rsyslog.d/*juju.conf")
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch juju log conf files")
	}

	backupFiles := []string{
		"/etc/init/juju-db.conf",
		filepath.Join(dataDir, "tools"),
		filepath.Join(dataDir, "server.pem"),
		filepath.Join(dataDir, "system-identity"),
		filepath.Join(dataDir, "nonce.txt"),
		filepath.Join(dataDir, "shared-secret"),
		"/home/ubuntu/.ssh/authorized_keys",
		"/var/log/juju/all-machines.log",
		"/var/log/juju/machine-0.log",
	}
	backupFiles = append(backupFiles, initMachineConfs...)
	backupFiles = append(backupFiles, agentConfs...)
	backupFiles = append(backupFiles, jujuLogConfs...)
	return backupFiles, nil
}

func dumpFiles(dumpdir string) error {
	tarFile, err := os.Create(filepath.Join(dumpdir, "root.tar"))
	if err != nil {
		return errors.Annotate(err, "error while opening initial archive")
	}
	defer tarFile.Close()
	backupFiles, err := getFilesToBackup()
	if err != nil {
		return errors.Annotate(err, "cannot determine files to backup")
	}
	_, err = tar.TarFiles(backupFiles, tarFile, sep)
	if err != nil {
		return errors.Annotate(err, "cannot backup configuration files")
	}
	return nil
}
