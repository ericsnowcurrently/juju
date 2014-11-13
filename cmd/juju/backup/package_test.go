// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/cmd/juju/backup"
	jujutesting "github.com/juju/juju/testing"
)

// MetaResultString is the expected output of running dumpMetadata() on
// s.metaresult.
var MetaResultString = `
backup ID:       "spam"
checksum:        ""
checksum format: ""
size (B):        0
stored:          0001-01-01 00:00:00 +0000 UTC
started:         0001-01-01 00:00:00 +0000 UTC
finished:        0001-01-01 00:00:00 +0000 UTC
notes:           ""
environment ID:  ""
machine ID:      ""
created on host: ""
juju version:    0.0.0
`[1:]

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

type BaseBackupSuite struct {
	jujutesting.FakeJujuHomeSuite

	command    *backup.Command
	metaresult *params.BackupMetadataResult
	data       string

	filename string
}

func (s *BaseBackupSuite) SetUpTest(c *gc.C) {
	s.FakeJujuHomeSuite.SetUpTest(c)

	s.command = backup.NewCommand().(*backup.Command)
	s.metaresult = &params.BackupMetadataResult{
		ID: "spam",
	}
	s.data = "<compressed archive data>"
}

func (s *BaseBackupSuite) TearDownTest(c *gc.C) {
	if s.filename != "" {
		err := os.Remove(s.filename)
		if !os.IsNotExist(err) {
			c.Check(err, gc.IsNil)
		}
	}

	s.FakeJujuHomeSuite.TearDownTest(c)
}

func (s *BaseBackupSuite) patchAPIClient(client backup.APIClient) {
	s.PatchValue(backup.NewAPIClient,
		func(c *backup.CommandBase) (backup.APIClient, error) {
			return client, nil
		},
	)
}

func (s *BaseBackupSuite) setSuccess() *fakeAPIClient {
	client := &fakeAPIClient{metaresult: s.metaresult}
	s.patchAPIClient(client)
	return client
}

func (s *BaseBackupSuite) setFailure(failure string) *fakeAPIClient {
	client := &fakeAPIClient{err: errors.New(failure)}
	s.patchAPIClient(client)
	return client
}

func (s *BaseBackupSuite) setDownload() *fakeAPIClient {
	client := s.setSuccess()
	client.archive = ioutil.NopCloser(bytes.NewBufferString(s.data))
	return client
}

func (s *BaseBackupSuite) checkArchive(c *gc.C) {
	c.Assert(s.filename, gc.Not(gc.Equals), "")
	archive, err := os.Open(s.filename)
	c.Assert(err, gc.IsNil)
	defer archive.Close()

	data, err := ioutil.ReadAll(archive)
	c.Check(string(data), gc.Equals, s.data)
}

func (s *BaseBackupSuite) diffStrings(c *gc.C, value, expected string) {
	// If only Go had a diff library.
	vlines := strings.Split(value, "\n")
	elines := strings.Split(expected, "\n")
	vsize := len(vlines)
	esize := len(elines)

	if vsize < 2 || esize < 2 {
		return
	}

	smaller := elines
	if vsize < esize {
		smaller = vlines
	}

	for i, _ := range smaller {
		vline := vlines[i]
		eline := elines[i]
		if vline != eline {
			c.Log("first mismatched line:")
			c.Log("expected: " + eline)
			c.Log("got:      " + vline)
			break
		}
	}

}

func (s *BaseBackupSuite) checkString(c *gc.C, value, expected string) {
	if !c.Check(value, gc.Equals, expected) {
		s.diffStrings(c, value, expected)
	}
}

func (s *BaseBackupSuite) checkStd(c *gc.C, ctx *cmd.Context, out, err string) {
	c.Check(ctx.Stdin.(*bytes.Buffer).String(), gc.Equals, "")
	s.checkString(c, ctx.Stdout.(*bytes.Buffer).String(), out)
	s.checkString(c, ctx.Stderr.(*bytes.Buffer).String(), err)
}

type fakeAPIClient struct {
	metaresult *params.BackupMetadataResult
	archive    io.ReadCloser
	err        error

	calls []string
	args  []string
	idArg string
	notes string
}

func (f *fakeAPIClient) Check(c *gc.C, id, notes string, calls ...string) {
	c.Check(f.calls, jc.DeepEquals, calls)
	c.Check(f.idArg, gc.Equals, id)
	c.Check(f.notes, gc.Equals, notes)
}

func (c *fakeAPIClient) Create(notes string) (*params.BackupMetadataResult, error) {
	c.calls = append(c.calls, "Create")
	c.args = append(c.args, "notes")
	c.notes = notes
	if c.err != nil {
		return nil, c.err
	}
	return c.metaresult, nil
}

func (c *fakeAPIClient) Info(id string) (*params.BackupMetadataResult, error) {
	c.calls = append(c.calls, "Info")
	c.args = append(c.args, "id")
	c.idArg = id
	if c.err != nil {
		return nil, c.err
	}
	return c.metaresult, nil
}

func (c *fakeAPIClient) List() (*params.BackupListResult, error) {
	c.calls = append(c.calls, "List")
	if c.err != nil {
		return nil, c.err
	}
	var result params.BackupListResult
	result.List = []params.BackupMetadataResult{*c.metaresult}
	return &result, nil
}

func (c *fakeAPIClient) Download(id string) (io.ReadCloser, error) {
	c.calls = append(c.calls, "Download")
	c.args = append(c.args, "id")
	c.idArg = id
	if c.err != nil {
		return nil, c.err
	}
	return c.archive, nil
}

func (c *fakeAPIClient) Remove(id string) error {
	c.calls = append(c.calls, "Remove")
	c.args = append(c.args, "id")
	c.idArg = id
	if c.err != nil {
		return c.err
	}
	return nil
}

func (c *fakeAPIClient) Close() error {
	return nil
}
