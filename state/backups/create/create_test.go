// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package create_test

import (
	"bytes"
	"io/ioutil"
	"os"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/state/backups"
	"github.com/juju/juju/state/backups/create"
	backupstesting "github.com/juju/juju/state/backups/testing"
)

type createSuite struct {
	LegacySuite
}

var _ = gc.Suite(&createSuite{}) // Register the suite.

type TestDBDumper struct {
	DumpDir string
}

func (d *TestDBDumper) Dump(dumpDir string) error {
	d.DumpDir = dumpDir
	return nil
}

func (s *createSuite) TestLegacy(c *gc.C) {
	meta := backupstesting.NewMetadataStarted()
	metadataFile, err := meta.AsJSONBuffer()
	c.Assert(err, jc.ErrorIsNil)
	_, testFiles, expected := s.createTestFiles(c)

	dumper := &TestDBDumper{}
	args := create.NewTestCreateArgs(testFiles, dumper, metadataFile)
	result, err := create.Create(args)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result, gc.NotNil)

	c.Assert(result.Archive, gc.NotNil)

	// Check the result.
	file, ok := result.Archive.(*os.File)
	c.Assert(ok, jc.IsTrue)

	s.checkSize(c, file, result.Size)
	s.checkChecksum(c, file, result.Checksum)
	s.checkArchive(c, file, expected)
}

func (s *createSuite) TestMetadataFileMissing(c *gc.C) {
	var testFiles []string
	dumper := &TestDBDumper{}

	args := create.NewTestCreateArgs(testFiles, dumper, nil)
	_, err := create.Create(args)

	c.Check(err, gc.ErrorMatches, "missing metadataReader")
}

func (s *createSuite) TestCreateOkay(c *gc.C) {

	// Patch the internals.
	archiveFile := ioutil.NopCloser(bytes.NewBufferString("<compressed tarball>"))
	expected := &backups.CreateResult{archiveFile, 10, "<checksum>"}
	received, testCreate := create.NewTestCreate(expected)
	s.PatchValue(create.RunCreate, testCreate)

	rootDir := "<was never set>"
	s.PatchValue(create.TestGetFilesToBackUp, func(root string, paths *create.Paths) ([]string, error) {
		rootDir = root
		return []string{"<some file>"}, nil
	})

	var receivedDBInfo *create.DBInfo
	s.PatchValue(create.GetDBDumper, func(info *create.DBInfo) (create.DBDumper, error) {
		receivedDBInfo = info
		return nil, nil
	})

	// Run the backup.
	paths := create.Paths{DataDir: "/var/lib/juju"}
	targets := set.NewStrings("juju", "admin")
	dbInfo := create.DBInfo{"a", "b", "c", targets}
	creator := create.NewCreator(&paths, &dbInfo)
	meta := backupstesting.NewMetadataStarted()
	backupstesting.SetOrigin(meta, "<env ID>", "<machine ID>", "<hostname>")
	meta.Notes = "some notes"
	result, err := creator.Create(meta)
	c.Assert(err, jc.IsNil)

	// Test the mocked values.
	filesToBackUp, _ := create.ExposeCreateArgs(received)
	c.Check(filesToBackUp, jc.SameContents, []string{"<some file>"})

	c.Check(receivedDBInfo.Address, gc.Equals, "a")
	c.Check(receivedDBInfo.Username, gc.Equals, "b")
	c.Check(receivedDBInfo.Password, gc.Equals, "c")
	c.Check(receivedDBInfo.Targets, gc.DeepEquals, targets)

	c.Check(rootDir, gc.Equals, "")
	c.Check(result, gc.Equals, expected)

	// Check the resulting metadata.
	c.Check(meta.ID(), gc.Equals, "")
	c.Check(meta.Size(), gc.Equals, int64(0))
	c.Check(meta.Checksum(), gc.Equals, "")
	c.Check(meta.Stored(), jc.IsNil)
	c.Check(meta.Origin.Environment, gc.Equals, "<env ID>")
	c.Check(meta.Origin.Machine, gc.Equals, "<machine ID>")
	c.Check(meta.Origin.Hostname, gc.Equals, "<hostname>")
	c.Check(meta.Notes, gc.Equals, "some notes")
}

func (s *createSuite) TestCreateFailToListFiles(c *gc.C) {
	s.PatchValue(create.TestGetFilesToBackUp, func(root string, paths *create.Paths, oldmachine string) ([]string, error) {
		return nil, errors.New("failed!")
	})
	meta := backupstesting.NewMetadataStarted()
	creator := create.NewCreator(nil, nil)
	_, err := creator.Create(meta)

	c.Check(err, gc.ErrorMatches, "while listing files to back up: failed!")
}

func (s *createSuite) TestCreateFailToCreate(c *gc.C) {
	s.PatchValue(create.TestGetFilesToBackUp, func(root string, paths *create.Paths, oldmachine string) ([]string, error) {
		return []string{}, nil
	})
	s.PatchValue(create.RunCreate, create.NewTestCreateFailure("failed!"))

	meta := backupstesting.NewMetadataStarted()
	creator := create.NewCreator(nil, nil)
	_, err := creator.Create(meta)

	c.Check(err, gc.ErrorMatches, "while creating backup archive: failed!")
}
