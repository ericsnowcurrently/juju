// Copyright 2013,2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup_test

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/state/backup"
	backuptesting "github.com/juju/juju/state/backup/testing"
)

type backupSuite struct {
	backuptesting.BaseSuite

	api backup.Backups
}

var _ = gc.Suite(&backupSuite{}) // Register the suite.

func (s *backupSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)

	s.api = backup.NewBackups(s.Storage)
}

func (s *backupSuite) setStored(id string) *time.Time {
	s.Storage.ID = id
	s.Storage.Meta = backuptesting.NewMetadataStarted(id, "")
	stored := time.Now().UTC()
	s.Storage.Meta.SetStored(&stored)
	return &stored
}

type fakeDumper struct{}

func (*fakeDumper) Dump(dumpDir string) error {
	return nil
}

func (s *backupSuite) checkFailure(c *gc.C, expected string) {
	s.PatchValue(backup.GetDBDumper, func(backup.DBInfo) (backup.DBDumper, error) {
		return &fakeDumper{}, nil
	})

	paths := backup.Paths{DataDir: "/var/lib/juju"}
	connInfo := backup.DBConnInfo{"a", "b", "c"}
	targets := set.NewStrings("juju", "admin")
	dbInfo := backup.DBInfo{connInfo, targets}
	origin := backup.NewOrigin("<env ID>", "<machine ID>", "<hostname>")
	_, err := s.api.Create(paths, dbInfo, origin, "some notes")

	c.Check(err, gc.ErrorMatches, expected)
}

func (s *backupSuite) TestNewBackups(c *gc.C) {
	api := backup.NewBackups(s.Storage)

	c.Check(api, gc.NotNil)
}

func (s *backupSuite) TestCreateOkay(c *gc.C) {

	// Patch the internals.
	archiveFile := ioutil.NopCloser(bytes.NewBufferString("<compressed tarball>"))
	result := backup.NewTestCreateResult(archiveFile, 10, "<checksum>")
	received, testCreate := backup.NewTestCreate(result)
	s.PatchValue(backup.RunCreate, testCreate)

	rootDir := "<was never set>"
	s.PatchValue(backup.TestGetFilesToBackUp, func(root string, paths backup.Paths) ([]string, error) {
		rootDir = root
		return []string{"<some file>"}, nil
	})

	var receivedDBInfo *backup.DBInfo
	s.PatchValue(backup.GetDBDumper, func(info backup.DBInfo) (backup.DBDumper, error) {
		receivedDBInfo = &info
		return nil, nil
	})

	stored := s.setStored("spam")

	// Run the backup.
	paths := backup.Paths{DataDir: "/var/lib/juju"}
	connInfo := backup.DBConnInfo{"a", "b", "c"}
	targets := set.NewStrings("juju", "admin")
	dbInfo := backup.DBInfo{connInfo, targets}
	origin := backup.NewOrigin("<env ID>", "<machine ID>", "<hostname>")
	meta, err := s.api.Create(paths, dbInfo, origin, "some notes")

	// Test the call values.
	s.Storage.CheckCalled(c, "spam", meta, archiveFile, "Add", "Metadata")
	filesToBackUp, _ := backup.ExposeCreateArgs(received)
	c.Check(filesToBackUp, jc.SameContents, []string{"<some file>"})

	err = receivedDBInfo.Validate()
	c.Assert(err, gc.IsNil)
	c.Check(receivedDBInfo.Address, gc.Equals, "a")
	c.Check(receivedDBInfo.Username, gc.Equals, "b")
	c.Check(receivedDBInfo.Password, gc.Equals, "c")
	c.Check(receivedDBInfo.Targets, gc.DeepEquals, targets)

	c.Check(rootDir, gc.Equals, "")

	// Check the resulting metadata.
	c.Check(meta, gc.Equals, s.Storage.MetaArg)
	c.Check(meta.ID(), gc.Equals, "spam")
	c.Check(meta.Size(), gc.Equals, int64(10))
	c.Check(meta.Checksum(), gc.Equals, "<checksum>")
	c.Check(meta.Stored().Unix(), gc.Equals, stored.Unix())
	c.Check(meta.Origin.Environment, gc.Equals, "<env ID>")
	c.Check(meta.Origin.Machine, gc.Equals, "<machine ID>")
	c.Check(meta.Origin.Hostname, gc.Equals, "<hostname>")
	c.Check(meta.Notes, gc.Equals, "some notes")

	// Check the file storage.
	s.Storage.Meta = meta
	s.Storage.File = archiveFile
	storedMeta, storedFile, err := s.Storage.Get(meta.ID())
	c.Check(err, gc.IsNil)
	c.Check(storedMeta, gc.DeepEquals, meta)
	data, err := ioutil.ReadAll(storedFile)
	c.Assert(err, gc.IsNil)
	c.Check(string(data), gc.Equals, "<compressed tarball>")
}

func (s *backupSuite) TestCreateFailToListFiles(c *gc.C) {
	s.PatchValue(backup.TestGetFilesToBackUp, func(root string, paths backup.Paths) ([]string, error) {
		return nil, errors.New("failed!")
	})

	s.checkFailure(c, "while listing files to back up: failed!")
}

func (s *backupSuite) TestCreateFailToCreate(c *gc.C) {
	s.PatchValue(backup.TestGetFilesToBackUp, func(root string, paths backup.Paths) ([]string, error) {
		return []string{}, nil
	})
	s.PatchValue(backup.RunCreate, backup.NewTestCreateFailure("failed!"))

	s.checkFailure(c, "while creating backup archive: failed!")
}

func (s *backupSuite) TestCreateFailToFinishMeta(c *gc.C) {
	s.PatchValue(backup.TestGetFilesToBackUp, func(root string, paths backup.Paths) ([]string, error) {
		return []string{}, nil
	})
	_, testCreate := backup.NewTestCreate(nil)
	s.PatchValue(backup.RunCreate, testCreate)
	s.PatchValue(backup.FinishMeta, backup.NewTestMetaFinisher("failed!"))

	s.checkFailure(c, "while updating metadata: failed!")
}

func (s *backupSuite) TestCreateFailToStoreArchive(c *gc.C) {
	s.PatchValue(backup.TestGetFilesToBackUp, func(root string, paths backup.Paths) ([]string, error) {
		return []string{}, nil
	})
	_, testCreate := backup.NewTestCreate(nil)
	s.PatchValue(backup.RunCreate, testCreate)
	s.PatchValue(backup.FinishMeta, backup.NewTestMetaFinisher(""))
	s.PatchValue(backup.StoreArchiveRef, backup.NewTestArchiveStorer("failed!"))

	s.checkFailure(c, "while storing backup archive: failed!")
}

func (s *backupSuite) TestStoreArchive(c *gc.C) {
	stored := s.setStored("spam")

	meta := backuptesting.NewMetadataStarted("", "")
	c.Assert(meta.ID(), gc.Equals, "")
	c.Assert(meta.Stored(), gc.IsNil)
	archive := &bytes.Buffer{}
	err := backup.StoreArchive(s.Storage, meta, archive)
	c.Assert(err, gc.IsNil)

	s.Storage.CheckCalled(c, "spam", meta, archive, "Add", "Metadata")
	c.Assert(meta.ID(), gc.Equals, "spam")
	c.Assert(meta.Stored(), jc.DeepEquals, stored)
}
