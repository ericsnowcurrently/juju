// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/juju/errors"
)

// TODO(ericsnow) lp-1392876
// Pull these from authoritative sources (see juju/paths, etc.).
const (
	machinesConfs = "jujud-machine-*.conf"
	agentsDir     = "agents"
	agentsConfs   = "machine-*"
	jujuInitConfs = "juju-*.conf"
	loggingConfs  = "*juju.conf"
	toolsDir      = "tools"

	sshIdentFile   = "system-identity"
	nonceFile      = "nonce.txt"
	allMachinesLog = "all-machines.log"
	machineLog     = "machine-%s.log"
	authKeysFile   = "authorized_keys"

	dbStartupConf = "juju-db.conf"
	dbPEM         = "server.pem"
	dbSecret      = "shared-secret"
)

// Paths holds the paths that backups needs.
type Paths struct {
	DataDir        string // /var/lib/juju
	InitDir        string // /etc/init
	LoggingConfDir string // /etc/rsyslog.d
	LogsDir        string // /var/log/juju
	SSHDir         string // /home/ubuntu/.ssh
}

// GetFilesToBackUp returns the paths that should be included in the
// backup archive.
func GetFilesToBackUp(rootDir string, paths *Paths, oldmachine string) ([]string, error) {
	var glob string

	glob = filepath.Join(rootDir, paths.InitDir, machinesConfs)
	initMachineConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch machine init files")
	}

	glob = filepath.Join(rootDir, paths.InitDir, jujuInitConfs)
	initConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch startup conf files")
	}

	glob = filepath.Join(rootDir, paths.DataDir, agentsDir, agentsConfs)
	agentConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch agent config files")
	}

	glob = filepath.Join(rootDir, paths.LoggingConfDir, loggingConfs)
	jujuLogConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch juju log conf files")
	}

	backupFiles := []string{
		filepath.Join(rootDir, paths.DataDir, toolsDir),

		filepath.Join(rootDir, paths.DataDir, sshIdentFile),

		filepath.Join(rootDir, paths.DataDir, dbPEM),
		filepath.Join(rootDir, paths.DataDir, dbSecret),
	}
	backupFiles = append(backupFiles, initMachineConfs...)
	backupFiles = append(backupFiles, agentConfs...)
	backupFiles = append(backupFiles, initConfs...)
	backupFiles = append(backupFiles, jujuLogConfs...)

	// Handle logs (might not exist).
	// TODO(ericsnow) We should consider dropping these entirely.
	allmachines := filepath.Join(rootDir, paths.LogsDir, allMachinesLog)
	if _, err := os.Stat(allmachines); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
		logger.Errorf("skipping missing file %q", allmachines)
	} else {
		backupFiles = append(backupFiles, allmachines)
	}
	// TODO(ericsnow) It might not be machine 0...
	machinelog := filepath.Join(rootDir, paths.LogsDir, fmt.Sprintf(machineLog, oldmachine))
	if _, err := os.Stat(machinelog); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
		logger.Errorf("skipping missing file %q", machinelog)
	} else {
		backupFiles = append(backupFiles, machinelog)
	}

	// Handle nonce.txt (might not exist).
	nonce := filepath.Join(rootDir, paths.DataDir, nonceFile)
	if _, err := os.Stat(nonce); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
		logger.Errorf("skipping missing file %q", nonce)
	} else {
		backupFiles = append(backupFiles, nonce)
	}

	// Handle user SSH files (might not exist).
	SSHDir := filepath.Join(rootDir, paths.SSHDir)
	if _, err := os.Stat(SSHDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
		logger.Errorf("skipping missing dir %q", SSHDir)
	} else {
		backupFiles = append(backupFiles, filepath.Join(SSHDir, authKeysFile))
	}

	return backupFiles, nil
}
