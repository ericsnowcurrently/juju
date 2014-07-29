// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/juju/utils/hash"
	"github.com/juju/utils/tar"
)

// This is effectively the "connection string".
// It probably doesn't belong here in the long term.
type DBConnInfo struct {
	Hostname string
	Username string
	Password string
}

type BackupCreator interface {
	Name() string
	Filename() string
	Timestamp() time.Time
	Prepare() error
	CleanUp() error
	Run(*DBConnInfo) (checksum string, err error)
}

//---------------------------
// BackupCreator implementation

type newBackup struct {
	tempdir  string
	root     string
	dumpdir  string
	archive  io.WriteCloser
	filename string
}

func (nb *newBackup) Name() string {
	return filepath.Base(nb.filename)
}

func (nb *newBackup) Filename() string {
	return nb.filename
}

func (nb *newBackup) Timestamp() time.Time {
	timestamp, tsErr := ExtractTimestamp(nb.Name())
	if tsErr != nil {
		timestamp = time.Now().UTC()
	}
	return timestamp
}

func (nb *newBackup) Prepare() error {
	// Prepare the temp directories.
	var bkpDir, dumpDir string
	tempDir, err := ioutil.TempDir("", "jujuBackup")
	if err == nil {
		logger.Debugf("backup temp dir: %s", tempDir)
		bkpDir = filepath.Join(tempDir, "juju-backup")
		dumpDir = filepath.Join(bkpDir, "dump")
		err = os.MkdirAll(dumpDir, os.FileMode(0755))
	}
	if err != nil {
		return fmt.Errorf("error creating backup temp directory: %v", err)
	}
	nb.tempdir = tempDir
	nb.root = bkpDir
	nb.dumpdir = dumpDir

	// Prepare an empty file into which to store the backup.
	archive, filename, err := CreateEmptyFile(tempDir+sep, 0600, false)
	if err != nil {
		return err
	}
	nb.archive = archive
	nb.filename = filename
	return nil
}

func (nb newBackup) CleanUp() error {
	if nb.tempdir != "" {
		err := os.RemoveAll(nb.tempdir)
		if err != nil {
			logger.Warningf("error removing tempdir (%s): %v", nb.tempdir, err)
		}
	}
	if nb.archive != nil {
		err := nb.archive.Close()
		if err != nil {
			return fmt.Errorf("error closing backup file: %v", err)
		}
	}
	return nil
}

func (nb newBackup) dumpDatabase(dbinfo *DBConnInfo) error {
	logger.Debugf("dumping database")

	err := dumpDatabase(dbinfo, nb.dumpdir)
	if err != nil {
		return fmt.Errorf("error dumping database: %v", err)
	}
	return nil
}

func (nb newBackup) bundleStateFiles() error {
	logger.Debugf("bundling state files")

	tarFile := filepath.Join(nb.root, "root.tar")
	archive, err := os.Create(tarFile)
	if err != nil {
		return fmt.Errorf("error creating temp tar file: %v", err)
	}

	backupFiles, err := getFilesToBackup()
	if err != nil {
		return fmt.Errorf("could not determine files to backup: %v", err)
	}

	_, err = tar.TarFiles(backupFiles, archive, sep)
	if err != nil {
		return fmt.Errorf("could not back up state files: %v", err)
	}
	return nil
}

func (nb newBackup) writeTarball() (string, error) {
	logger.Debugf("writing backup tarball: %s", nb.filename)

	// Create the tarball.
	hasher := hash.NewSHA1Proxy(nb.archive)
	tarball := gzip.NewWriter(hasher)
	// We ignore the checksum returned here since it matches the
	// uncompressed archive rather than the compressed one.
	_, err := tar.TarFiles([]string{nb.root}, tarball, nb.tempdir+sep)

	// Close the tarball.
	tbErr := tarball.Close()
	if tbErr != nil && err == nil {
		err = fmt.Errorf("error closing gzip writer: %v", tbErr)
	}

	sha1sum := hasher.Hash() // ...must happen *after* closing the tarball.
	return sha1sum, err
}

func (nb *newBackup) Run(dbinfo *DBConnInfo) (string, error) {
	err := nb.dumpDatabase(dbinfo)
	if err != nil {
		return "", err
	}

	err = nb.bundleStateFiles()
	if err != nil {
		return "", err
	}

	sha1sum, err := nb.writeTarball()
	if err != nil {
		return "", err
	}

	return sha1sum, nil
}
