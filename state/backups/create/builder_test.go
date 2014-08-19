// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package create_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/backups/config"
	"github.com/juju/juju/state/backups/create"
	"github.com/juju/juju/state/backups/testing"
)

// Register the suite.
var _ = gc.Suite(&createSuite{})

type createSuite struct {
	testing.BackupSuite

	ranCommand bool
}

func (s *createSuite) patchExternals(c *gc.C) {
	s.PatchValue(create.GetDumpCmd, func(
		config config.BackupsConfig, outdir string,
	) (string, []string, error) {
		_, args, err := config.DBDump(outdir)
		c.Assert(err, gc.IsNil)
		return "bogusmongodump", args, nil
	})
	//	s.PatchValue(create.GetFilesToBackup, func(
	//		config config.BackupsConfig,
	//	) ([]string, error) {
	//		return s.testFiles, nil
	//	})
	s.PatchValue(create.RunCommand, func(cmd string, args ...string) error {
		s.ranCommand = true
		return nil
	})
}

func (s *createSuite) checkArchive(c *gc.C, filename string, paths config.Paths) {
	ar := archive.NewArchive(filename, "")

	compressed, err := os.Open(filename)
	c.Assert(err, gc.IsNil)
	defer compressed.Close()

	uncompressed, err = gzip.NewReader(file)
	c.Assert(err, gc.IsNil)

	tarReader := tar.NewReader(gzr)

	// Iterate through the files in the archive.
	foundDBDumpDir := false
	foundContentDir := false
	var fileBundle io.ReadWriteCloser
	dbDumpFiles := []string{}
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		c.Assert(err, gc.IsNil)

		if hdr.Name == ar.ContentDir() {
			foundContentDir = true
		} else if hdr.Name == ar.DBDumpDir() {
			foundDBDumpeDir = true
		} else if strings.HasPrefix(hdr.Name, ar.DBDumpDir()) {
			dbDumpFiles.append(hdr.Name)
		} else if hdr.Name == ar.BundleFile() {
			_, fileBundle, _ = s.WS.Create("")
			_, err := io.Copy(fileBundle, tarReader)
			c.Assert(err, gc.IsNil)
			_, err = fileBundle.Seek(0, os.SEEK_SET)
			c.Assert(err, gc.IsNil)
		}

		buf, err := ioutil.ReadAll(tr)
		c.Assert(err, gc.IsNil)
		tarContents[hdr.Name] = string(buf)
	}
	for _, expectedContent := range expected {
		fullExpectedContent := strings.TrimPrefix(expectedContent.Name, string(os.PathSeparator))
		body, ok := tarContents[fullExpectedContent]
		c.Log(tarContents)
		c.Log(expected)
		c.Log(fmt.Sprintf("checking for presence of %q on tar file", fullExpectedContent))
		c.Assert(ok, gc.Equals, true)
		if expectedContent.Body != "" {
			c.Log("Also checking the file contents")
			c.Assert(body, gc.Equals, expectedContent.Body)
		}
	}

}

func (s *createSuite) checkResult(c *gc.C, result *create.Result) {
	c.Assert(result, gc.NotNil)

	filename := s.CopyArchiveFile(c, result.ArchiveFile)

	fileShaSum := shaSumFile(c, filename)
	c.Assert(result.Checksum, gc.Equals, fileShaSum)

	ar := archive.NewArchive(filename, s.WS.RootDir())

	s.checkArchive(ar, s.Paths)

	bkpExpectedContents := []expectedTarContents{
		{"juju-backup", ""},
		{"juju-backup/dump", ""},
		{"juju-backup/root.tar", ""},
	}
	s.assertTarContents(c, bkpExpectedContents, filename, true)
}

func (s *createSuite) TestBuilderBuildOkay(c *gc.C) {
	s.createTestFiles(c)
	s.patchExternals(c)

	builder, err := create.NewBuilder(s.Config)
	c.Assert(err, gc.IsNil)
	defer builder.Close()

	result, err := builder.Build()
	c.Check(err, gc.IsNil)

	c.Assert(s.ranCommand, gc.Equals, true)
	s.checkResult(c, result)
}
