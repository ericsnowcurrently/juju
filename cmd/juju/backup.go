// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"fmt"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/juju/cmd/envcmd"
	"github.com/juju/juju/juju"
	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/state/backup"
)

type backupClient interface {
	Backup(backupFilePath string, validate bool) (string, error)
}

type BackupCommand struct {
	envcmd.EnvCommandBase
	Verify   bool
	Filename string
}

var backupDoc = fmt.Sprintf(`
This command will generate a backup of juju's state as a local gzipped tar
file (.tar.gz).  If no filename is provided, one is generated with a name
like this:

  %s

where <timestamp> is the timestamp of when the backup is generated.

The filename of the generated archive will be printed upon success.
`, fmt.Sprintf(backup.FilenameTemplate, "<timestamp>"))

func (c *BackupCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "backup",
		Args:    "[--verify] [filename]",
		Purpose: "create a backup of juju's state",
		Doc:     strings.TrimSpace(backupDoc),
	}
}

func (c *BackupCommand) Init(args []string) error {
	// XXX Add --verify flag.
	filename, err := cmd.ZeroOrOneArgs(args)
	if err == nil {
		c.Filename = filename
	}
	return err
}

func (c *BackupCommand) runBackup(ctx *cmd.Context, client backupClient) error {
	filename, err := client.Backup(c.Filename, c.Verify)
	if params.IsCodeNotImplemented(err) {
		// XXX Handle this in client instead (at direct point of interaction)?
		return fmt.Errorf(backupAPIIncompatibility)
	} else if err != nil {
		return err
	}

	fmt.Fprintln(ctx.Stdout, filename)
	return nil
}

const backupAPIIncompatibility = "server version not compatible for backup " +
	"with client version"

func (c *BackupCommand) Run(ctx *cmd.Context) error {
	client, err := juju.NewAPIClientFromName(c.EnvName)
	if err != nil {
		return err
	}
	defer client.Close()

	return c.runBackup(ctx, client)
}
