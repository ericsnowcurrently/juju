// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/juju/loggo"
	"github.com/juju/utils"

	"github.com/juju/juju/version"
)

var (
	logger = loggo.GetLogger("juju.state.backup")
	sep    = string(os.PathSeparator)
)

// CreateBackup creates a new backup archive.  If store is true, the
// backup is stored to environment storage.
// The backup contains a dump folder with the output of mongodump command
// and a root.tar file which contains all the system files obtained from
// the output of getFilesToBackup.
func CreateBackup(
	dbinfo *DBConnInfo, name string, stor BackupStorage, creator BackupCreator,
) (*BackupInfo, error) {
	logger.Infof("backing up juju state")
	info := BackupInfo{Name: name}

	archive, err := create(&info, dbinfo, creator)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	logger.Infof("created: %q (SHA-1: %s)", info.Name, info.CheckSum)

	// Store the backup.
	logger.Debugf("storing %q", info.Name)
	err = stor.Add(&info, archive)
	if err != nil {
		return nil, fmt.Errorf("error storing archive: %v", err)
	}
	logger.Infof("stored: %q (SHA-1: %s)", info.Name, info.CheckSum)

	return &info, err
}

// Wraps the creator and sets the info fields.
func create(info *BackupInfo, dbinfo *DBConnInfo, creator BackupCreator) (_ io.ReadCloser, err error) {
	if creator == nil {
		creator = &newBackup{}
	}
	cleanup := func() {
		nbErr := creator.CleanUp()
		if nbErr != nil && err == nil {
			err = nbErr
		}
	}

	// Prepare the backup.
	err = creator.Prepare()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Run the backup.
	sha1sum, err := creator.Run(dbinfo)
	if err != nil {
		return nil, err
	}

	// Set the backup fields.
	if info.Name == "" {
		info.Name = creator.Name()
	}
	info.Timestamp = creator.Timestamp()
	info.CheckSum = sha1sum
	info.Version = version.Current.Number

	// Open the archive (before we delete it).
	archive, err := os.Open(creator.Filename())
	if err != nil {
		return nil, fmt.Errorf("error opening archive: %v", err)
	}
	finfo, err := archive.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %v", err)
	}
	info.Size = finfo.Size()

	return archive, err
}

//---------------------------
// legacy API

type fileStore struct {
	dirname string
}

func (f *fileStore) Put(name string, r io.Reader, len int64) error {
	stored, err := os.Create(filepath.Join(f.dirname, name))
	if err != nil {
		return err
	}
	defer stored.Close()
	if _, err = io.Copy(stored, r); err != nil {
		return err
	}
	return nil
}

func (f *fileStore) Get(string) (io.ReadCloser, error)                      { return nil, nil }
func (f *fileStore) List(string) ([]string, error)                          { return nil, nil }
func (f *fileStore) URL(string) (string, error)                             { return "", nil }
func (f *fileStore) DefaultConsistencyStrategy() (as utils.AttemptStrategy) { return }
func (f *fileStore) ShouldRetry(error) bool                                 { return false }
func (f *fileStore) Remove(string) error                                    { return nil }
func (f *fileStore) RemoveAll() error                                       { return nil }

// XXX Remove!
func Backup(pw, tag, outputFolder, host string) (string, string, error) {
	dbinfo := DBConnInfo{
		Hostname: host,
		Username: tag,
		Password: pw,
	}
	stor, err := NewBackupStorage(nil, &fileStore{outputFolder})
	if err != nil {
		return "", "", err
	}
	backup, err := CreateBackup(&dbinfo, "", stor, nil)
	if err != nil {
		return "", "", err
	}
	return backup.Name, backup.CheckSum, nil
}
