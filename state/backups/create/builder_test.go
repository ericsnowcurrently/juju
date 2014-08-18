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
	"github.com/juju/juju/testing"
)

func shaSumFile(c *gc.C, fileToSum string) string {
	f, err := os.Open(fileToSum)
	c.Assert(err, gc.IsNil)
	defer f.Close()
	shahash := sha1.New()
	_, err = io.Copy(shahash, f)
	c.Assert(err, gc.IsNil)
	return base64.StdEncoding.EncodeToString(shahash.Sum(nil))
}

// Register the suite.
var _ = gc.Suite(&createSuite{})

type createSuite struct {
	testing.BaseSuite
	cwd        string
	config     config.BackupsConfig
	testFiles  []string
	ranCommand bool
}

func (s *createSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
	s.cwd = c.MkDir()

	// Set the config.
	connInfo := config.NewDBConnInfo("localhost:8080", "bogus-user", "boguspw")
	dbInfo, err := config.NewDBInfo(connInfo)
	c.Assert(err, gc.IsNil)
	config, err := config.NewBackupsConfig(dbInfo, nil)
	c.Assert(err, gc.IsNil)
	s.config = config
}

func (s *createSuite) createTestFiles(c *gc.C) {
	tarDirE := path.Join(s.cwd, "TarDirectoryEmpty")
	err := os.Mkdir(tarDirE, os.FileMode(0755))
	c.Check(err, gc.IsNil)

	tarDirP := path.Join(s.cwd, "TarDirectoryPopulated")
	err = os.Mkdir(tarDirP, os.FileMode(0755))
	c.Check(err, gc.IsNil)

	tarSubFile1 := path.Join(tarDirP, "TarSubFile1")
	tarSubFile1Handle, err := os.Create(tarSubFile1)
	c.Check(err, gc.IsNil)
	tarSubFile1Handle.WriteString("TarSubFile1")
	tarSubFile1Handle.Close()

	tarSubDir := path.Join(tarDirP, "TarDirectoryPopulatedSubDirectory")
	err = os.Mkdir(tarSubDir, os.FileMode(0755))
	c.Check(err, gc.IsNil)

	tarFile1 := path.Join(s.cwd, "TarFile1")
	tarFile1Handle, err := os.Create(tarFile1)
	c.Check(err, gc.IsNil)
	tarFile1Handle.WriteString("TarFile1")
	tarFile1Handle.Close()

	tarFile2 := path.Join(s.cwd, "TarFile2")
	tarFile2Handle, err := os.Create(tarFile2)
	c.Check(err, gc.IsNil)
	tarFile2Handle.WriteString("TarFile2")
	tarFile2Handle.Close()
	s.testFiles = []string{tarDirE, tarDirP, tarFile1, tarFile2}

}

func (s *createSuite) patchExternals(c *gc.C) {
	s.PatchValue(create.GetDumpCmd, func(
		config config.BackupsConfig, outdir string,
	) (string, []string, error) {
		_, args, err := config.DBDump(outdir)
		c.Assert(err, gc.IsNil)
		return "bogusmongodump", args, nil
	})
	s.PatchValue(create.GetFilesToBackup, func(
		config config.BackupsConfig,
	) ([]string, error) {
		return s.testFiles, nil
	})
	s.PatchValue(create.RunCommand, func(cmd string, args ...string) error {
		s.ranCommand = true
		return nil
	})
}

func (s *createSuite) copyArchiveFile(c *gc.C, orig io.ReadCloser) string {
	defer orig.Close()

	bkpFile := "juju-backup.tar.gz"
	filename := path.Join(s.cwd, bkpFile)
	file, err := os.Create(filename)
	c.Assert(err, gc.IsNil)
	defer file.Close()

	_, err = io.Copy(file, orig)
	c.Assert(err, gc.IsNil)

	return filename
}

type expectedTarContents struct {
	Name string
	Body string
}

var testExpectedTarContents = []expectedTarContents{
	{"TarDirectoryEmpty", ""},
	{"TarDirectoryPopulated", ""},
	{"TarDirectoryPopulated/TarSubFile1", "TarSubFile1"},
	{"TarDirectoryPopulated/TarDirectoryPopulatedSubDirectory", ""},
	{"TarFile1", "TarFile1"},
	{"TarFile2", "TarFile2"},
}

// Assert thar contents checks that the tar[.gz] file provided contains the
// Expected files
// expectedContents: is a slice of the filenames with relative paths that are
// expected to be on the tar file
// tarFile: is the path of the file to be checked
func (s *createSuite) assertTarContents(
	c *gc.C, expected []expectedTarContents, tarFile string, compressed bool,
) {
	f, err := os.Open(tarFile)
	c.Assert(err, gc.IsNil)
	defer f.Close()
	var r io.Reader = f
	if compressed {
		r, err = gzip.NewReader(r)
		c.Assert(err, gc.IsNil)
	}

	tr := tar.NewReader(r)

	tarContents := make(map[string]string)
	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		c.Assert(err, gc.IsNil)
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

	filename := s.copyArchiveFile(c, result.ArchiveFile)

	fileShaSum := shaSumFile(c, filename)
	c.Assert(result.Checksum, gc.Equals, fileShaSum)

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

	builder, err := create.NewBuilder(s.config)
	c.Assert(err, gc.IsNil)
	defer builder.Close()

	result, err := builder.Build()
	c.Check(err, gc.IsNil)

	c.Assert(s.ranCommand, gc.Equals, true)
	s.checkResult(c, result)
}
