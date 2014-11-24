// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import (
	"os"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/state/backups"
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
	args := backups.NewTestCreateArgs(testFiles, dumper, metadataFile)
	result, err := backups.Create(args)
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

	args := backups.NewTestCreateArgs(testFiles, dumper, nil)
	_, err := backups.Create(args)

	c.Check(err, gc.ErrorMatches, "missing metadataReader")
}

func (s *createSuite) TestCreateFailToListFiles(c *gc.C) {
	s.PatchValue(backups.TestGetFilesToBackUp, func(root string, paths *backups.Paths, oldmachine string) ([]string, error) {
		return nil, errors.New("failed!")
	})
	meta := backupstesting.NewMetadataStarted()
	creator := backups.NewCreator(nil, nil)
	_, err := creator.Create(meta)

	c.Check(err, gc.ErrorMatches, "while listing files to back up: failed!")
}

func (s *createSuite) TestCreateFailToCreate(c *gc.C) {
	s.PatchValue(backups.TestGetFilesToBackUp, func(root string, paths *backups.Paths, oldmachine string) ([]string, error) {
		return []string{}, nil
	})
	s.PatchValue(backups.RunCreate, backups.NewTestCreateFailure("failed!"))

	meta := backupstesting.NewMetadataStarted()
	creator := backups.NewCreator(nil, nil)
	_, err := creator.Create(meta)

	c.Check(err, gc.ErrorMatches, "while creating backup archive: failed!")
}
