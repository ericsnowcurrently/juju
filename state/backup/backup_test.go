// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup_test

import (
	"io"
	"os"
	"path/filepath"

	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/backup"
)

type fileStore struct {
	dirname string
}

func (f *fileStore) Put(name string, r io.Reader, len int64) error {
	filename := filepath.Join(f.dirname, name)
	err := os.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return err
	}
	stored, err := os.Create(filename)
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

//---------------------------
// tests

func (b *BackupSuite) TestBackup(c *gc.C) {
	b.createTestFiles(c)
	ranCommand := false

	b.PatchValue(backup.GetMongodumpPath, func() (string, error) {
		return "bogusmongodump", nil
	})
	b.PatchValue(backup.GetFilesToBackup, func() ([]string, error) {
		return b.testFiles, nil
	})
	b.PatchValue(backup.DoBackup, func(command string, args ...string) error {
		ranCommand = true
		return nil
	})

	dbinfo := backup.DBConnInfo{
		Hostname: "localhost:8080",
		Username: "bogus-user",
		Password: "boguspassword",
	}
	stor, err := backup.NewBackupStorage(nil, &fileStore{b.cwd})
	c.Assert(err, gc.IsNil)
	backup, err := backup.CreateBackup(&dbinfo, "", stor, nil)
	c.Assert(err, gc.IsNil)
	c.Check(ranCommand, gc.Equals, true)

	bkpFile, shaSum := backup.Name, backup.CheckSum

	// It is important that the filename uses non-special characters
	// only because it is returned in a header (unencoded) by the
	// backup API call. This also avoids compatibility problems with
	// client side filename conventions.
	c.Check(bkpFile, gc.Matches, `^[a-z0-9_.-]+$`)

	filename := filepath.Join(b.cwd, "backups", bkpFile, "archive")
	fileShaSum := shaSumFile(c, filename)
	c.Check(shaSum, gc.Equals, fileShaSum)

	bkpExpectedContents := []expectedTarContents{
		{"juju-backup", ""},
		{"juju-backup/dump", ""},
		{"juju-backup/root.tar", ""},
	}
	b.checkTarContents(c, bkpExpectedContents, filename, true)
}
