// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"os"
	"path/filepath"

	"github.com/juju/testing"
	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	jujuutils "github.com/juju/juju/utils"
	"github.com/juju/juju/utils/archives"
)

// FilesSuite is used for file-oriented testing.
// It facilitates the following when embedded in a gocheck suite type:
// - logger redirect
// - no outgoing network access
// - scrubbing of env vars
type FilesSuite struct {
	testing.IsolationSuite
}

func (s *FilesSuite) SetUpSuite(c *gc.C) {
	s.IsolationSuite.SetUpSuite(c)
	s.PatchValue(&utils.OutgoingAccessAllowed, false)
}

func (s *FilesSuite) CreateTempFile(
	c *gc.C, dirname, basename string,
) (
	*os.File, string,
) {
	if dirname == "" {
		dirname = c.MkDir()
	}
	if basename == "" {
		randname, err := jujuutils.RandomHex(12)
		c.Assert(err, gc.IsNil)
		basename = "tmp-" + randname
	}
	filename := filepath.Join(dirname, basename)

	file, err := os.Create(filename)
	c.Assert(err, gc.IsNil)
	s.AddCleanup(func(*gc.C) {
		os.Remove(filename)
	})

	return file, filename
}

func (s *FilesSuite) WriteTempFile(
	c *gc.C, data, dirname, basename string,
) string {
	file, filename := s.CreateTempFile(c, dirname, basename)
	defer file.Close()

	_, err := file.Write([]byte(data))
	c.Assert(err, gc.IsNil)
	return filename
}

func (s *FilesSuite) CreateGzipFile(
	c *gc.C, data, dirname, basename string,
) (
	string, string,
) {
	compressed, err := archives.GzipData(data)
	c.Assert(err, gc.IsNil)
	filename := s.WriteTempFile(c, compressed, dirname, basename)
	return filename, compressed
}
