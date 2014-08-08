// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/juju/utils/filestorage"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/backups"
	"github.com/juju/juju/state/backups/config"
	"github.com/juju/juju/testing"
)

type CreateSuite struct {
	testing.BaseSuite
	cwd       string
	testFiles []string
}

func (s *CreateSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
	s.cwd = c.MkDir()
}

func (s *CreateSuite) createTestFiles(c *gc.C) {
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
func (s *CreateSuite) assertTarContents(
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

var _ = gc.Suite(&createSuite{})

type createSuite struct {
	CreateSuite
}

func (s *createSuite) TestCreateLegacy(c *gc.C) {
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

	conf, err := config.NewBackupsConfigRaw(
		"localhost:8080",
		"bogus-user",
		"boguspassword",
	)
	c.Assert(err, gc.IsNil)
	metadata := filestorage.NewMetadata(nil)

	archive, err := backups.Create(conf, metadata)
	c.Check(err, gc.IsNil)
	c.Assert(ranCommand, gc.Equals, true)

	bkpFile := "juju-backup.tar.gz"
	filename := path.Join(s.cwd, bkpFile)
	func() {
		defer archive.Close()
		file, err := os.Create(filename)
		c.Assert(err, gc.IsNil)
		defer file.Close()
		io.Copy(file, archive)
	}()

	shaSum := metadata.Checksum()
	fileShaSum := shaSumFile(c, filename)
	c.Assert(shaSum, gc.Equals, fileShaSum)

	bkpExpectedContents := []expectedTarContents{
		{"juju-backup", ""},
		{"juju-backup/dump", ""},
		{"juju-backup/root.tar", ""},
	}
	s.assertTarContents(c, bkpExpectedContents, filename, true)
}
