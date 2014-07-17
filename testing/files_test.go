// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing_test

import (
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/testing"
)

var _ = gc.Suite(&TestingFilesSuite{})

type TestingFilesSuite struct {
	testing.FilesSuite
	tempfiles []string
}

func (s *TestingFilesSuite) SetUpSuite(c *gc.C) {
	s.FilesSuite.SetUpSuite(c)
}

func (s *TestingFilesSuite) TearDownSuite(c *gc.C) {
	s.FilesSuite.TearDownSuite(c)

	// Make sure all the temp files were deleted.
	for _, filename := range s.tempfiles {
		_, err := os.Stat(filename)
		c.Check(os.IsNotExist(err), gc.Equals, true)
	}
}

func (s *TestingFilesSuite) TestCreateTempFileDefaults(c *gc.C) {
	file, filename := s.CreateTempFile(c, "", "")
	s.tempfiles = append(s.tempfiles, filename)

	c.Check(file, gc.NotNil)
	c.Check(filename, gc.Not(gc.Equals), "")
}

func (s *TestingFilesSuite) TestCreateTempFileName(c *gc.C) {
	_, filename := s.CreateTempFile(c, "/tmp", "juju-file-testing-tempfile")
	s.tempfiles = append(s.tempfiles, filename)
	expected := filepath.Join("/tmp", "juju-file-testing-tempfile")

	c.Check(filename, gc.Equals, expected)
}

func (s *TestingFilesSuite) TestCreateTempFileSame(c *gc.C) {
	file, filename := s.CreateTempFile(c, "/tmp", "juju-file-testing-tempfile")
	s.tempfiles = append(s.tempfiles, filename)
	c.Assert(file, gc.NotNil)
	fileinfo, err := file.Stat()
	c.Assert(err, gc.IsNil)

	c.Check(fileinfo.Name(), gc.Equals, filepath.Base(filename))
}

func (s *TestingFilesSuite) TestCreateTempFileExists(c *gc.C) {
	filename := filepath.Join("/tmp", "juju-file-testing-tempfile")
	file, _ := s.CreateTempFile(c, "/tmp", "juju-file-testing-tempfile")
	s.tempfiles = append(s.tempfiles, filename)
	c.Check(file, gc.NotNil)
	file.Close()

	_, err := os.Stat(filename)
	c.Check(err, gc.IsNil)
}

func (s *TestingFilesSuite) TestWriteTempfile(c *gc.C) {
	data := "spam"
	filename := s.WriteTempFile(c, data, "", "")
	s.tempfiles = append(s.tempfiles, filename)

	raw, err := ioutil.ReadFile(filename)
	c.Assert(err, gc.IsNil)
	c.Check(string(raw), gc.Equals, data)
}

func (s *TestingFilesSuite) TestCreateGzipFile(c *gc.C) {
	data := "spam"
	filename, compressed := s.CreateGzipFile(c, data, "", "")
	s.tempfiles = append(s.tempfiles, filename)

	// Check the raw compressed file.
	gzraw, err := ioutil.ReadFile(filename)
	c.Assert(err, gc.IsNil)
	c.Check(string(gzraw), gc.Equals, compressed)

	// Check the compressed data.
	file, err := os.Open(filename)
	c.Assert(err, gc.IsNil)
	gzfile, err := gzip.NewReader(file)
	c.Assert(err, gc.IsNil)

	buffer := make([]byte, len(data)*2, len(data)*2)
	size, err := gzfile.Read(buffer)
	c.Assert(err, gc.IsNil)
	c.Check(string(buffer[0:size]), gc.Equals, data)
}
