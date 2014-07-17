// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// XXX Change to backup_test
package backup

import (
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/testing"
	"github.com/juju/juju/utils/archives"
)

func (b *BackupSuite) TestGetHash(c *gc.C) {
	archive := testing.NewCloseableBufferString("bam")
	hash, err := GetHash(archive)

	c.Assert(err, gc.IsNil)
	c.Check(hash, gc.Equals, "evJYWUtQ/4dKBHtUqSRC6B9FjPs=")
}

func (b *BackupSuite) TestUncompressAndGetHash(c *gc.C) {
	data := "bam"
	compressed, err := archives.GzipData(data)
	c.Assert(err, gc.IsNil)
	archive := testing.NewCloseableBufferString(compressed)
	hash, err := UncompressAndGetHash(archive, "application/x-tar-gz")

	c.Assert(err, gc.IsNil)
	c.Check(hash, gc.Equals, "evJYWUtQ/4dKBHtUqSRC6B9FjPs=")
}

func (b *BackupSuite) TestUncompressAndGetHashMimetypeFallback(c *gc.C) {
	data := "bam"
	compressed, err := archives.GzipData(data)
	c.Assert(err, gc.IsNil)
	archive := testing.NewCloseableBufferString(compressed)
	hash, err := UncompressAndGetHash(archive, "totally not a mimetype")

	c.Assert(err, gc.IsNil)
	c.Check(hash, gc.Equals, "evJYWUtQ/4dKBHtUqSRC6B9FjPs=")
}

func (b *BackupSuite) TestUncompressAndGetHashNotCompressed(c *gc.C) {
	_, err := UncompressAndGetHash(nil, "application/x-tar")

	c.Assert(err, gc.ErrorMatches, "not compressed: application/x-tar")
}

func (b *BackupSuite) TestGetHashDefault(c *gc.C) {
	data := "bam"
	filename, _ := b.CreateGzipFile(c, data, "", "")
	hash, err := GetHashDefault(filename)

	c.Assert(err, gc.IsNil)
	c.Check(hash, gc.Equals, "evJYWUtQ/4dKBHtUqSRC6B9FjPs=")
}

func (b *BackupSuite) TestGetHashDefaultFileMissing(c *gc.C) {
	filename := "this file does not exist!"
	_, err := GetHashDefault(filename)

	c.Assert(err, gc.ErrorMatches, "could not open file: .*")
}

func (b *BackupSuite) TestGetHashDefaultFileEmpty(c *gc.C) {
	filename := b.WriteTempFile(c, "", "", "")
	_, err := GetHashDefault(filename)

	c.Assert(err, gc.ErrorMatches, "file is empty")
}

func (b *BackupSuite) TestGetHashByFilename(c *gc.C) {
	data := "bam"
	filename, _ := b.CreateGzipFile(c, data, "", "somefile.tar.gz")
	hash, err := GetHashByFilename(filename)

	c.Assert(err, gc.IsNil)
	c.Check(hash, gc.Equals, "evJYWUtQ/4dKBHtUqSRC6B9FjPs=")
}

func (b *BackupSuite) TestGetHashByFilenameFileMissing(c *gc.C) {
	filename := "this file does not exist!"
	_, err := GetHashByFilename(filename)

	c.Assert(err, gc.ErrorMatches, "could not open file: .*")
}

func (b *BackupSuite) TestGetHashByFilenameFileEmpty(c *gc.C) {
	filename := b.WriteTempFile(c, "", "", "")
	_, err := GetHashByFilename(filename)

	c.Assert(err, gc.ErrorMatches, "file is empty")
}

func (b *BackupSuite) TestGetHashByFilenameBadMimetype(c *gc.C) {
	data := "bam"
	filename := b.WriteTempFile(c, data, "", "somefile.not-a-mime-type")
	_, err := GetHashByFilename(filename)

	c.Assert(err, gc.ErrorMatches, "unsupported filename (.*)")
}

func (b *BackupSuite) TestGetHashByFilenameStandardMimetype(c *gc.C) {
	data := "bam"
	filename, _ := b.CreateGzipFile(c, data, "", "somefile.html")
	hash, err := GetHashByFilename(filename)

	c.Assert(err, gc.IsNil)
	c.Check(hash, gc.Equals, "evJYWUtQ/4dKBHtUqSRC6B9FjPs=")
}

func (b *BackupSuite) TestGetHashByFilenameTarfile(c *gc.C) {
	data := "bam"
	filename := b.WriteTempFile(c, data, "", "somefile.tar")
	hash, err := GetHashByFilename(filename)

	c.Assert(err, gc.IsNil)
	c.Check(hash, gc.Equals, "evJYWUtQ/4dKBHtUqSRC6B9FjPs=")
}
