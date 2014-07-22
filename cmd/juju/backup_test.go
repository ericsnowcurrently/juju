// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/juju/cmd"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/state/backup"
	"github.com/juju/juju/testing"
)

type BackupCommandSuite struct {
	testing.FakeJujuHomeSuite
}

var _ = gc.Suite(&BackupCommandSuite{})

//---------------------------
// a fake client and command

var fakeBackupFilename = fmt.Sprintf(backup.FilenameTemplate, "999999999")

type fakeBackupClient struct {
	hash    string
	expHash string
	err     *params.Error
}

func (c *fakeBackupClient) Backup(filename string, excl bool) (string, string, string, *params.Error) {
	if c.err != nil {
		return "", "", "", c.err
	}

	if filename == "" {
		filename = fakeBackupFilename
	}

	_, err := os.Stat(filename)
	if err == nil {
		// The file exists!
		if excl {
			err := params.Error{
				Message: fmt.Sprintf("already exists: %s", filename),
			}
			return "", "", "", &err
		}
	}

	// Don't actually do anything!
	return filename, c.hash, c.expHash, nil
}

type fakeBackupCommand struct {
	BackupCommand
	client fakeBackupClient
}

func (c *fakeBackupCommand) Run(ctx *cmd.Context) error {
	return c.runBackup(ctx, &c.client)
}

//---------------------------
// help tests

// We could generalize these tests for all commands...

func (s *BackupCommandSuite) TestBackupeHelp(c *gc.C) {
	// Check the help output.
	info := (&BackupCommand{}).Info()
	_usage := fmt.Sprintf("usage: juju %s [options] %s", info.Name, info.Args)
	_purpose := fmt.Sprintf("purpose: %s", info.Purpose)

	// Run the command, ensuring it is actually there.
	out := badrun(c, 0, "backup", "--help")
	out = out[:len(out)-1] // Strip the trailing \n.

	// Check the usage string.
	parts := strings.SplitN(out, "\n", 2)
	usage := parts[0]
	out = parts[1]
	c.Check(usage, gc.Equals, _usage)

	// Check the purpose string.
	parts = strings.SplitN(out, "\n\n", 2)
	purpose := parts[0]
	out = parts[1]
	c.Check(purpose, gc.Equals, _purpose)

	// Check the options.
	parts = strings.SplitN(out, "\n\n", 2)
	options := strings.Split(parts[0], "\n-")
	out = parts[1]
	c.Check(options, gc.HasLen, 4)
	c.Check(options[0], gc.Equals, "options:")
	c.Check(strings.Contains(options[1], "environment"), gc.Equals, true)

	// Check the doc.
	doc := out
	c.Check(doc, gc.Equals, info.Doc)
}

//---------------------------
// options tests

func (s *BackupCommandSuite) TestBackupDefaultsValid(c *gc.C) {
	client := fakeBackupClient{}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command)

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, fakeBackupFilename+"\n")
}

func (s *BackupCommandSuite) TestBackupDefaultsExists(c *gc.C) {
	filename := filepath.Join(c.MkDir(), "backup.tar.gz")
	file, err := os.Create(filename)
	c.Assert(err, gc.IsNil)
	err = file.Close()
	c.Assert(err, gc.IsNil)

	client := fakeBackupClient{}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, filename)

	c.Check(err, gc.ErrorMatches, "already exists: .*")
	c.Check(testing.Stdout(ctx), gc.Matches, "")
	c.Check(testing.Stderr(ctx), gc.Matches, "")
}

func (s *BackupCommandSuite) TestBackupDefaultsHashMatch(c *gc.C) {
	client := fakeBackupClient{hash: "abc", expHash: "abc"}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command)

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, fakeBackupFilename+"\n")
}

func (s *BackupCommandSuite) TestBackupDefaultsHashMismatch(c *gc.C) {
	client := fakeBackupClient{hash: "abc", expHash: "cba"}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command)

	c.Check(err, gc.ErrorMatches, `SHA-1 checksum mismatch \(archive still saved\)`)
	c.Check(testing.Stdout(ctx), gc.Matches, fakeBackupFilename+"\n")
	c.Check(testing.Stderr(ctx), gc.Matches, "")
}

func (s *BackupCommandSuite) TestBackupDefaultsHashMissing(c *gc.C) {
	client := fakeBackupClient{hash: "abc", expHash: ""}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command)

	c.Check(err, gc.ErrorMatches, `SHA-1 checksum mismatch \(archive still saved\)`)
	c.Check(testing.Stdout(ctx), gc.Matches, fakeBackupFilename+"\n")
	c.Check(testing.Stderr(ctx), gc.Matches, "")
}

func (s *BackupCommandSuite) TestBackupOverwriteExists(c *gc.C) {
	filename := filepath.Join(c.MkDir(), "backup.tar.gz")
	file, err := os.Create(filename)
	c.Assert(err, gc.IsNil)
	err = file.Close()
	c.Assert(err, gc.IsNil)

	client := fakeBackupClient{}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, "--overwrite", filename)

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, filename+"\n")
}

func (s *BackupCommandSuite) TestBackupOverwriteNoExists(c *gc.C) {
	filename := filepath.Join(c.MkDir(), "backup.tar.gz")

	client := fakeBackupClient{}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, "--overwrite", filename)

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, filename+"\n")
}

func (s *BackupCommandSuite) TestBackupNoVerifyMatch(c *gc.C) {
	client := fakeBackupClient{hash: "abc", expHash: "abc"}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, "--noverify")

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, fmt.Sprintf(backup.FilenameTemplate, ".*")+"\n")
}

func (s *BackupCommandSuite) TestBackupNoVerifyNoMatch(c *gc.C) {
	client := fakeBackupClient{hash: "abc", expHash: "cba"}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, "--noverify")

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, fmt.Sprintf(backup.FilenameTemplate, ".*")+"\n")
}

func (s *BackupCommandSuite) TestBackupNoVerifyMissing(c *gc.C) {
	client := fakeBackupClient{hash: "abc", expHash: ""}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, "--noverify")

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, fmt.Sprintf(backup.FilenameTemplate, ".*")+"\n")
}

func (s *BackupCommandSuite) TestBackupFilename(c *gc.C) {
	filename := "foo.tgz"
	client := fakeBackupClient{}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, filename)

	c.Check(err, gc.IsNil)
	c.Check(testing.Stdout(ctx), gc.Matches, filename+"\n")
}

//---------------------------
// failure tests

func (s *BackupCommandSuite) TestBackupError(c *gc.C) {
	client := fakeBackupClient{
		err: &params.Error{"something went wrong", ""},
	}
	command := fakeBackupCommand{client: client}

	_, err := testing.RunCommand(c, &command)

	c.Check(err, gc.NotNil)
}

func (s *BackupCommandSuite) TestBackupAPIIncompatibility(c *gc.C) {
	failure := params.Error{
		Message: "bogus request",
		Code:    params.CodeNotImplemented,
	}
	client := fakeBackupClient{err: &failure}
	command := fakeBackupCommand{client: client}

	_, err := testing.RunCommand(c, &command)

	c.Check(err, gc.ErrorMatches, "server version not compatible with client version")
}
