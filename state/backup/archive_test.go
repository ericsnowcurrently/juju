// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// XXX Change to backup_test
package backup

import (
	"fmt"
	"path"

	gc "launchpad.net/gocheck"
)

func (b *BackupSuite) TestCreateArchiveUncompressed(c *gc.C) {
	b.createTestFiles(c)
	outputTar := path.Join(b.cwd, "output_tar_file.tar")
	trimPath := fmt.Sprintf("%s/", b.cwd)
	shaSum, err := CreateArchive(b.testFiles, outputTar, trimPath, false)
	c.Check(err, gc.IsNil)
	fileShaSum, err := GetHashByFilename(outputTar)
	c.Assert(err, gc.IsNil)
	c.Assert(shaSum, gc.Equals, fileShaSum)
	b.removeTestFiles(c)
	b.assertTarContents(c, testExpectedTarContents, outputTar, false)
}

func (b *BackupSuite) TestCreateArchiveCompressed(c *gc.C) {
	b.createTestFiles(c)
	outputTarGz := path.Join(b.cwd, "output_tar_file.tgz")
	trimPath := fmt.Sprintf("%s/", b.cwd)
	shaSum, err := CreateArchive(b.testFiles, outputTarGz, trimPath, true)
	c.Check(err, gc.IsNil)

	fileShaSum, err := GetHashByFilename(outputTarGz)
	c.Assert(err, gc.IsNil)
	c.Assert(shaSum, gc.Equals, fileShaSum)

	b.assertTarContents(c, testExpectedTarContents, outputTarGz, true)
}
