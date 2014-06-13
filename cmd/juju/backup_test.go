// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"errors"
	"fmt"
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

// a fake client and command

var fakeBackupFilename = fmt.Sprintf(backup.FilenameTemplate, "999999999")

type fakeBackupClient struct {
	err error
}

func (c *fakeBackupClient) Backup(backupFilePath string, verify bool) (string, error) {
	if c.err != nil {
		return "", c.err
	}

	if backupFilePath == "" {
		backupFilePath = fakeBackupFilename
	}
	/* Don't actually do anything! */
	return backupFilePath, c.err
}

type fakeBackupCommand struct {
	BackupCommand
	client fakeBackupClient
}

func (c *fakeBackupCommand) Run(ctx *cmd.Context) error {
	return c.runBackup(ctx, &c.client)
}

// help tests

// XXX Generalize for all commands?
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
	c.Assert(usage, gc.Equals, _usage)

	// Check the purpose string.
	parts = strings.SplitN(out, "\n\n", 2)
	purpose := parts[0]
	out = parts[1]
	c.Assert(purpose, gc.Equals, _purpose)

	// Check the options.
	parts = strings.SplitN(out, "\n\n", 2)
	options := strings.Split(parts[0], "\n-")
	out = parts[1]
	c.Assert(options, gc.HasLen, 2)
	c.Assert(options[0], gc.Equals, "options:")
	c.Assert(strings.Contains(options[1], "environment"), gc.Equals, true)

	// Check the doc.
	doc := out
	c.Assert(doc, gc.Equals, info.Doc)
}

// options tests

func (s *BackupCommandSuite) TestBackupDefaults(c *gc.C) {
	client := fakeBackupClient{}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command)

	c.Assert(err, gc.IsNil)
	c.Assert(testing.Stdout(ctx), gc.Matches, fakeBackupFilename+"\n")
}

func (s *BackupCommandSuite) TestBackupFilename(c *gc.C) {
	filename := "foo.tgz"
	client := fakeBackupClient{}
	command := fakeBackupCommand{client: client}

	ctx, err := testing.RunCommand(c, &command, filename)

	c.Assert(err, gc.IsNil)
	c.Assert(testing.Stdout(ctx), gc.Matches, filename+"\n")
}

// failure tests

func (s *BackupCommandSuite) TestBackupError(c *gc.C) {
	client := fakeBackupClient{err: errors.New("something went wrong")}
	command := fakeBackupCommand{client: client}

	_, err := testing.RunCommand(c, &command)

	c.Assert(err, gc.NotNil)
}

func (s *BackupCommandSuite) TestBackupAPIIncompatibility(c *gc.C) {
	failure := params.Error{
		Message: "bogus request",
		Code:    params.CodeNotImplemented,
	}
	client := fakeBackupClient{err: &failure}
	command := fakeBackupCommand{client: client}

	_, err := testing.RunCommand(c, &command)

	c.Assert(err, gc.ErrorMatches, backupAPIIncompatibility)
}
