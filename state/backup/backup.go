// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("juju.backup")

var getFilesToBackup = _getFilesToBackup

func _getFilesToBackup() ([]string, error) {
	const dataDir string = "/var/lib/juju"
	initMachineConfs, err := filepath.Glob("/etc/init/jujud-machine-*.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch machine upstart files: %v", err)
	}
	agentConfs, err := filepath.Glob(filepath.Join(dataDir, "agents", "machine-*"))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent configuration files: %v", err)
	}
	jujuLogConfs, err := filepath.Glob("/etc/rsyslog.d/*juju.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch juju log conf files: %v", err)
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

var runCommand = _runCommand

func _runCommand(cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	out, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	if _, ok := err.(*exec.ExitError); ok && len(out) > 0 {
		return fmt.Errorf("error executing %q: %s", cmd, strings.Replace(string(out), "\n", "; ", -1))
	}
	return fmt.Errorf("cannot execute %q: %v", cmd, err)
}

var getMongodumpPath = _getMongodumpPath

func _getMongodumpPath() (string, error) {
	const mongoDumpPath string = "/usr/lib/juju/bin/mongodump"

	if _, err := os.Stat(mongoDumpPath); err == nil {
		return mongoDumpPath, nil
	}

	path, err := exec.LookPath("mongodump")
	if err != nil {
		return "", err
	}
	return path, nil
}

// Backup creates a tar.gz file named juju-backup_<date YYYYMMDDHHMMSS>.tar.gz
// in the specified outputFolder.
// The backup contains a dump folder with the output of mongodump command
// and a root.tar file which contains all the system files obtained from
// the output of getFilesToBackup
func Backup(password string, username string, outputFolder string, addr string) (string, string, error) {
	sep := string(os.PathSeparator)

	// YYYYMMDDHHMMSS
	formattedDate := time.Now().Format("20060102150405")

	bkpFile := fmt.Sprintf("juju-backup_%s.tar.gz", formattedDate)

	// Prepare the temp directories.
	tempDir, err := ioutil.TempDir("", "jujuBackup")
	defer os.RemoveAll(tempDir)
	bkpDir := filepath.Join(tempDir, "juju-backup")
	dumpDir := filepath.Join(bkpDir, "dump")
	err = os.MkdirAll(dumpDir, os.FileMode(0755))
	if err != nil {
		return "", "", fmt.Errorf("cannot create backup temporary directory: %v", err)
	}

	// Dump the database.
	mongodumpPath, err := getMongodumpPath()
	if err != nil {
		return "", "", fmt.Errorf("mongodump not available: %v", err)
	}
	err = runCommand(
		mongodumpPath,
		"--oplog",
		"--ssl",
		"--host", addr,
		"--username", username,
		"--password", password,
		"--out", dumpDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to dump database: %v", err)
	}

	// Tar up the various state-related files (not including the DB dump).
	tarFile := filepath.Join(bkpDir, "root.tar")
	backupFiles, err := getFilesToBackup()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine files to backup: %v", err)
	}
	_, err = tarFiles(backupFiles, tarFile, sep, false)
	if err != nil {
		return "", "", fmt.Errorf("cannot backup configuration files: %v", err)
	}

	// Create a new tarball containing the previous tarfile and the DB dump.
	shaSum, err := tarFiles([]string{bkpDir},
		filepath.Join(outputFolder, bkpFile),
		tempDir+sep,
		true)
	if err != nil {
		return "", "", fmt.Errorf("cannot tar configuration files: %v", err)
	}

	return bkpFile, shaSum, nil
}

// StorageName returns the path in environment storage where a backup
// should be stored.
func StorageName(filename string) string {
	// Use of path.Join instead of filepath.Join is intentional - this
	// is an environment storage path not a filesystem path.
	return path.Join("/backups", filepath.Base(filename))
}
