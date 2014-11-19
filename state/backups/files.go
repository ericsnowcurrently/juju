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
// Pull these from authoritative sources (see
// github.com/juju/juju/juju/paths, etc.):
const (
	dataDir        = "/var/lib/juju"
	startupDir     = "/etc/init"
	loggingConfDir = "/etc/rsyslog.d"
	logsDir        = "/var/log/juju"
	sshDir         = "/home/ubuntu/.ssh"

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
type Paths interface {
	// FilesToBackUp returns the paths that should be included in the
	// backup archive.
	FilesToBackUp() ([]string, error)
}

type LegacyPaths struct {
	RootDir    string
	DataDir    string
	LogsDir    string
	OldMachine string
}

func NewPaths(rootDir, oldMachine string) *LegacyPaths {
}

func (p *LegacyPaths) resolve(parts ...string) string {
	return filepath.Join(p.RootDir, parts...)
}

// FilesToBackUp returns the paths that should be included in the
// backup archive.
func (p *LegacyPaths) FilesToBackUp() ([]string, error) {
	var glob string

	glob = p.resolve(startupDir, machinesConfs)
	initMachineConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch machine init files")
	}

	glob = p.resolve(startupDir, jujuInitConfs)
	initConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch startup conf files")
	}

	glob = p.resolve(paths.DataDir, agentsDir, agentsConfs)
	agentConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch agent config files")
	}

	glob = p.resolve(loggingConfDir, loggingConfs)
	jujuLogConfs, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch juju log conf files")
	}

	backupFiles := []string{
		p.resolve(paths.DataDir, toolsDir),

		p.resolve(paths.DataDir, sshIdentFile),

		p.resolve(paths.DataDir, dbPEM),
		p.resolve(paths.DataDir, dbSecret),
	}
	backupFiles = append(backupFiles, initMachineConfs...)
	backupFiles = append(backupFiles, agentConfs...)
	backupFiles = append(backupFiles, initConfs...)
	backupFiles = append(backupFiles, jujuLogConfs...)

	// Handle logs (might not exist).
	// TODO(ericsnow) We should consider dropping these entirely.
	allmachines := p.resolve(paths.LogsDir, allMachinesLog)
	if _, err := os.Stat(allmachines); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
		logger.Errorf("skipping missing file %q", allmachines)
	} else {
		backupFiles = append(backupFiles, allmachines)
	}
	machinelog := p.resolve(paths.LogsDir, fmt.Sprintf(machineLog, p.OldMachine))
	if _, err := os.Stat(machinelog); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
		logger.Errorf("skipping missing file %q", machinelog)
	} else {
		backupFiles = append(backupFiles, machinelog)
	}

	// Handle nonce.txt (might not exist).
	nonce := p.resolve(paths.DataDir, nonceFile)
	if _, err := os.Stat(nonce); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
		logger.Errorf("skipping missing file %q", nonce)
	} else {
		backupFiles = append(backupFiles, nonce)
	}

	// Handle user SSH files (might not exist).
	SSHDir := p.resolve(sshDir)
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
