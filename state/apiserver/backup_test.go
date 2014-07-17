// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/juju/juju/rpc/errors"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/apiserver"
	jc "github.com/juju/testing/checkers"
	gc "launchpad.net/gocheck"
)

type backupSuite struct {
	authHttpSuite
}

var _ = gc.Suite(&backupSuite{})

func (s *backupSuite) SetUpSuite(c *gc.C) {
	s.authHttpSuite.SetUpSuite(c)
	s.APIBinding = "backup"
	s.HTTPMethod = "POST"
	s.Mimetype = "application/x-tar-gz"
}

func (s *backupSuite) TestBackupHTTPHandling(c *gc.C) {
	var result errors.APIError
	s.CheckServedSecurely(c)
	s.CheckHTTPMethodInvalid(c, "GET", &result)
	s.CheckHTTPMethodInvalid(c, "PUT", &result)

	s.CheckRequiresAuth(c, &result)
	s.CheckRequiresUser(c, &result)

	s.CheckLegacyPathUnavailable(c)
}

func (s *backupSuite) TestBackupCalledAndFileServed(c *gc.C) {
	testGetMongoConnectionInfo := func(thisState *state.State) *state.Info {
		info := &state.Info{
			Password: "foobar",
			Tag:      "machine-0",
		}
		info.Addrs = append(info.Addrs, "localhost:80")
		return info
	}
	var data struct{ tempDir, mongoPassword, username, address string }
	testBackup := func(password string, username string, tempDir string, address string) (string, string, error) {
		data.tempDir = tempDir
		data.mongoPassword = password
		data.username = username
		data.address = address
		backupFilePath := filepath.Join(tempDir, "testBackupFile")
		file, err := os.Create(backupFilePath)
		if err != nil {
			return "", "", err
		}
		file.Write([]byte("foobarbam"))
		file.Close()
		return backupFilePath, "some-sha", nil
	}
	s.PatchValue(&apiserver.Backup, testBackup)
	s.PatchValue(apiserver.GetMongoConnInfo, testGetMongoConnectionInfo)

	resp := s.SendValid(c)

	// Check the response.
	s.CheckFileResponse(c, resp, "foobarbam", "application/octet-stream")
	c.Check(resp.Header.Get("Digest"), gc.Equals, "SHA=some-sha")

	// Check the passed values.
	c.Check(data.tempDir, gc.NotNil)
	_, err := os.Stat(data.tempDir)
	c.Check(err, jc.Satisfies, os.IsNotExist)
	c.Check(data.mongoPassword, gc.Equals, "foobar")
	c.Check(data.username, gc.Equals, "machine-0")
	c.Check(data.address, gc.Equals, "localhost:80")
}

func (s *backupSuite) TestBackupErrorWhenBackupFails(c *gc.C) {
	var data struct{ tempDir string }
	testBackup := func(password string, username string, tempDir string, address string) (string, string, error) {
		data.tempDir = tempDir
		return "", "", fmt.Errorf("something bad")
	}
	s.PatchValue(&apiserver.Backup, testBackup)

	resp, err := s.Send(c)
	s.CheckStatusCode(c, resp, http.StatusInternalServerError)
	c.Check(err, gc.ErrorMatches, "backup failed: something bad")

	// Check the passed values.
	c.Assert(data.tempDir, gc.NotNil)
	_, err = os.Stat(data.tempDir)
	c.Check(err, jc.Satisfies, os.IsNotExist)
}

func (s *backupSuite) TestBackupErrorWhenBackupFileDoesNotExist(c *gc.C) {
	var data struct{ tempDir string }
	testBackup := func(password string, username string, tempDir string, address string) (string, string, error) {
		data.tempDir = tempDir
		backupFilePath := filepath.Join(tempDir, "testBackupFile")
		return backupFilePath, "some-sha", nil
	}
	s.PatchValue(&apiserver.Backup, testBackup)

	resp, err := s.Send(c)
	s.CheckStatusCode(c, resp, http.StatusInternalServerError)
	c.Check(err, gc.ErrorMatches, `backup failed \(invalid backup file\), could not open file: .*`)

	// Check the passed values.
	c.Assert(data.tempDir, gc.NotNil)
	_, err = os.Stat(data.tempDir)
	c.Check(err, jc.Satisfies, os.IsNotExist)
}

func (s *backupSuite) TestBackupErrorWhenBackupFileEmpty(c *gc.C) {
	var data struct{ tempDir string }
	testBackup := func(password string, username string, tempDir string, address string) (string, string, error) {
		data.tempDir = tempDir
		backupFilePath := filepath.Join(tempDir, "testBackupFile")
		file, err := os.Create(backupFilePath)
		c.Assert(err, gc.IsNil)
		file.Close()
		return backupFilePath, "some-sha", nil
	}
	s.PatchValue(&apiserver.Backup, testBackup)

	resp, err := s.Send(c)
	s.CheckStatusCode(c, resp, http.StatusInternalServerError)
	c.Check(err, gc.ErrorMatches, `backup failed \(invalid backup file\), file is empty`)

	// Check the passed values.
	c.Assert(data.tempDir, gc.NotNil)
	_, err = os.Stat(data.tempDir)
	c.Check(err, jc.Satisfies, os.IsNotExist)
}
