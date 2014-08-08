// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import (
	"path"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/backups"
	"github.com/juju/juju/state/backups/config"
)

var _ = gc.Suite(&legacySuite{})

type legacySuite struct {
	CreateSuite
}

func (s *legacySuite) TestBackup(c *gc.C) {
	s.createTestFiles(c)

	s.PatchValue(backups.GetDumpCmd, func(
		config config.BackupsConfig, outdir string,
	) (string, []string, error) {
		_, args, err := config.DBDump(outdir)
		c.Assert(err, gc.IsNil)
		return "bogusmongodump", args, nil
	})
	s.PatchValue(backups.GetFilesToBackup, func(
		config config.BackupsConfig,
	) ([]string, error) {
		return s.testFiles, nil
	})
	ranCommand := false
	s.PatchValue(backups.RunCommand, func(cmd string, args ...string) error {
		ranCommand = true
		return nil
	})

	bkpFile, shaSum, err := backups.Backup("boguspassword", "bogus-user", s.cwd, "localhost:8080")
	c.Check(err, gc.IsNil)
	c.Assert(ranCommand, gc.Equals, true)

	// It is important that the filename uses non-special characters
	// only because it is returned in a header (unencoded) by the
	// backup API call. This also avoids compatibility problems with
	// client side filename conventions.
	c.Check(bkpFile, gc.Matches, `^[a-z0-9_.-]+$`)

	fileShaSum := shaSumFile(c, path.Join(s.cwd, bkpFile))
	c.Assert(shaSum, gc.Equals, fileShaSum)

	bkpExpectedContents := []expectedTarContents{
		{"juju-backup", ""},
		{"juju-backup/dump", ""},
		{"juju-backup/root.tar", ""},
	}
	s.assertTarContents(c, bkpExpectedContents, path.Join(s.cwd, bkpFile), true)
}

func (s *legacySuite) TestStorageName(c *gc.C) {
	c.Check(backups.StorageName("foo"), gc.Equals, "/backups/foo")
	c.Check(backups.StorageName("/foo/bar"), gc.Equals, "/backups/bar")
	c.Check(backups.StorageName("foo/bar"), gc.Equals, "/backups/bar")
}
