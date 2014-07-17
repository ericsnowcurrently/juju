// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// XXX Change to backup_test
package backup

import (
	"time"

	gc "launchpad.net/gocheck"
)

func (b *BackupSuite) TestDefaultFilename(c *gc.C) {
	filename := DefaultFilename()

	// This is a sanity check that no one accidentally
	// (or accidentally maliciously) broken the default filename format.
	c.Check(filename, gc.Matches, `jujubackup-\d{8}-\d{6}\..*`)
	// The most crucial part is that the suffix is .tar.gz.
	c.Assert(filename, gc.Matches, `.*\.tar\.gz$`)
}

func (b *BackupSuite) TestDefaultFilenameDateFormat(c *gc.C) {
	filename := DefaultFilename()
	_, err := TimestampFromDefaultFilename(filename)

	c.Check(err, gc.IsNil)
}

func (b *BackupSuite) TestDefaultFilenameUnique(c *gc.C) {
	filename1 := DefaultFilename()
	time.Sleep(1 * time.Second)
	filename2 := DefaultFilename()

	c.Check(filename1, gc.Not(gc.Equals), filename2)
}
