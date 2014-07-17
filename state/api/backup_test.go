// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api_test

import (
	"fmt"
	//	"io/ioutil"
	"net/http"
	"os"
	//	"path/filepath"

	gc "launchpad.net/gocheck"

	//	simpleTesting "github.com/juju/juju/rpc/simple/testing"
	"github.com/juju/juju/state/api"
	"github.com/juju/juju/state/apiserver"
	"github.com/juju/juju/state/backup"
	"github.com/juju/juju/testing"
)

//const tempPrefix = "test-juju-backup-client-"

var _ = gc.Suite(&backupSuite{})

type backupSuite struct {
	authHttpSuite
	//	tempDir string
}

//func (s *backupSuite) TempDir(c *gc.C) string {
//    if s.tempDir == "" {
//        var err error
//	    s.tempDir, err = ioutil.TempDir("", tempPrefix)
//	    c.Assert(err, gc.IsNil)
//	    s.removeBackupTemp(s.tempDir)
//    }
//    return s.tempDir
//}

//---------------------------
// archive helpers

//func (s *backupSuite) writeTempArchive(c *gc.C, data, dirname string) string {
//	if dirname == "" {
//		dirname = s.TempDir(c)
//	}
//	filename := filepath.Join(dirname, "backup.tar.gz")
//    s.AddCleanup(func(*gc.C) { os.remove(filename) })
//
//	archive, err := os.Create(filename)
//	c.Assert(err, gc.IsNil)
//	defer archive.Close()
//
//	_, err = archive.Write([]byte(data))
//	c.Assert(err, gc.IsNil)
//	return filename
//}
//
//func (s *backupSuite) createArchive(c *gc.C, data, dirname string) (string, string) {
//	compressed, err := simpleTesting.GzipData(data)
//	c.Assert(err, gc.IsNil)
//	filename := s.writeTempArchive(c, compressed, dirname)
//	return filename, compressed
//}
//
//func (s *backupSuite) removeBackupTemp(filename string) {
//	s.AddCleanup(func(*gc.C) {
//        err := os.RemoveAll(filename)
//        if err != nil {
//            fmt.Printf("'%s' not removed: %v", filename, err)
//        }
//    })
//}

func (s *backupSuite) assertFilenameMatchesDefault(c *gc.C, filename string) {
	c.Assert(filename, gc.Matches, fmt.Sprintf(backup.FilenameTemplate, ".*"))
}

//---------------------------
// response helpers

func getBackupResponse(data string, digest string) *http.Response {
	resp := http.Response{}

	resp.Body = testing.NewCloseableBufferString(data)

	resp.Header = http.Header{}
	if digest == "" && data != "" {
		hash, _ := backup.UncompressAndGetHash(resp.Body, "application/x-tar-gz")
		resp.Header.Set("Digest", "SHA="+hash)
	} else {
		resp.Header.Set("Digest", digest)
	}

	return &resp
}

func (s *backupSuite) setBackupServerSuccess(c *gc.C, data string) {
	succeed := func(pw, user, dir, addr string) (string, string, error) {
		backupFilePath, _ := s.CreateGzipFile(c, data, dir, "")
		hash, err := backup.GetHashDefault(backupFilePath)
		c.Assert(err, gc.IsNil)

		return backupFilePath, hash, nil
	}
	s.PatchValue(&apiserver.Backup, succeed)
}

func (s *backupSuite) setBackupServerError(err error) {
	fail := func(pw, user, dir, addr string) (string, string, error) {
		return "", "", err
	}
	s.PatchValue(&apiserver.Backup, fail)
}

//---------------------------
// Success tests

func (s *backupSuite) TestBackupCustomFilename(c *gc.C) {
	s.setBackupServerSuccess(c, "foobarbaz")
	client := s.APIClient()

	filenameOrig := fmt.Sprintf(backup.FilenameTemplate, "20140623-010101")
	validate := false
	filename, digest, err := client.Backup(filenameOrig, validate)
	if filename != "" {
		os.Remove(filename)
	}
	c.Assert(err, gc.IsNil)

	c.Check(filename, gc.Equals, filenameOrig)
	c.Check(digest, gc.Equals, "")
}

func (s *backupSuite) TestBackupDefaultFilename(c *gc.C) {
	s.setBackupServerSuccess(c, "foobarbaz")
	client := s.APIClient()

	validate := false
	filename, digest, err := client.Backup("", validate)
	if filename != "" {
		os.Remove(filename)
	}
	c.Assert(err, gc.IsNil)

	s.assertFilenameMatchesDefault(c, filename)
	c.Check(digest, gc.Equals, "")
}

func (s *backupSuite) TestBackupValidateHashSuccess(c *gc.C) {
	hash := "kVi4iZOb1a7A9SBOPIv7ShWUKIU="
	data := "...archive data..."

	filename, compressed := s.CreateGzipFile(c, data, "", "")

	resp := getBackupResponse(compressed, "SHA="+hash)
	digest, err := api.ExtractBackupDigestFromHeader(resp)
	c.Assert(err, gc.IsNil)
	err = api.ValidateBackupHash(filename, digest)
	c.Assert(err, gc.IsNil)
}

//---------------------------
// Failure tests

func (s *backupSuite) TestBackupDigestHeaderMissing(c *gc.C) {
	resp := http.Response{}
	_, err := api.ExtractBackupDigestFromHeader(&resp)

	c.Assert(err, gc.ErrorMatches, "'SHA' digest missing from response")
}

func (s *backupSuite) TestBackupDigestHeaderEmpty(c *gc.C) {
	resp := getBackupResponse("", "")
	_, err := api.ExtractBackupDigestFromHeader(resp)

	c.Assert(err, gc.ErrorMatches, "'SHA' digest missing from response")
}

func (s *backupSuite) TestBackupDigestHeaderInvalid(c *gc.C) {
	resp := getBackupResponse("", "not valid SHA digest")
	_, err := api.ExtractBackupDigestFromHeader(resp)

	c.Assert(err, gc.ErrorMatches, "malformed digest: not valid SHA digest")
}

func (s *backupSuite) TestBackupOutfileNotOpened(c *gc.C) {
	resp := getBackupResponse("...archive data...", "")
	digest, err := api.ExtractBackupDigestFromHeader(resp)
	c.Assert(err, gc.IsNil)

	err = api.ValidateBackupHash("/tmp/backup-does-not-exist.tar.gz", digest)

	c.Assert(err, gc.ErrorMatches, "could not verify backup file: could not open file: .*")
}

func (s *backupSuite) TestBackupHashNotExtracted(c *gc.C) {
	gethash := func(filename string) (string, error) {
		return "", fmt.Errorf("could not extract hash: error!")
	}
	s.PatchValue(api.GetBackupHash, gethash)

	filename, _ := s.CreateGzipFile(c, "", "", "") // no data

	resp := getBackupResponse("...", "")
	digest, err := api.ExtractBackupDigestFromHeader(resp)

	err = api.ValidateBackupHash(filename, digest)

	c.Assert(err, gc.ErrorMatches, "could not verify backup file: could not extract hash: error!")
}

func (s *backupSuite) TestBackupHashMismatch(c *gc.C) {
	hash := "invalid hash"
	data := "...archive data..."

	filename, _ := s.CreateGzipFile(c, data, "", "")

	resp := getBackupResponse(data, "SHA="+hash)
	digest, err := api.ExtractBackupDigestFromHeader(resp)
	c.Assert(err, gc.IsNil)

	err = api.ValidateBackupHash(filename, digest)

	c.Assert(err, gc.ErrorMatches, "archive hash did not match value from server: .*")
}

//---------------------------
// System tests

/*
1) the filename is created on disk
2) The content of the filename is not nil
3) It is a valid tarball
4) The hash matches expectations (though presumably that's already covered by other code)
5) we could assert that some of the filenames in the tarball match what we expect to be in a backup.
*/

/*
// XXX How to test this?
func (s *backupSuite) TestBackup(c *gc.C) {
	client := s.APIState.Client()
	expected := fmt.Sprintf(backup.FilenameTemplate, "20140623-010101")

	validate := false
	filename, digest, err := client.Backup(expected, validate)
	c.Assert(err, gc.IsNil)
	c.Assert(filename, gc.Equals, expected)
}
*/
