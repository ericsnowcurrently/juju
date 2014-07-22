// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"fmt"
	"strings"

	"launchpad.net/gnuflag"

	"github.com/juju/cmd"
	"github.com/juju/juju/cmd/envcmd"
	"github.com/juju/juju/juju"
	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/state/backup"
)

type backupClient interface {
	Backup(string, bool) (string, string, string, *params.Error)
}

type BackupCommand struct {
	envcmd.EnvCommandBase
	VerifyHash bool
	Overwrite  bool
	Filename   string
	noverify   bool
}

var backupDoc = fmt.Sprintf(`
This command will generate a backup of juju's state as a local gzipped
tar file (.tar.gz).  If no filename is provided, one is generated with
a name like this:

  %s

where <timestamp> is the "YYYYMMDD-hhmmss" timestamp of when the backup
was generated.

The filename of the generated archive will be printed upon success.
`, fmt.Sprintf(backup.FilenameTemplate, "<timestamp>"))

func (c *BackupCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "backup",
		Args:    "[--noverify] [--overwrite] [filename]",
		Purpose: "create a backup of juju's state",
		Doc:     strings.TrimSpace(backupDoc),
	}
}

func (c *BackupCommand) SetFlags(f *gnuflag.FlagSet) {
	c.EnvCommandBase.SetFlags(f)
	f.BoolVar(&c.noverify, "noverify", false, "do not verify the checksum")
	f.BoolVar(&c.Overwrite, "overwrite", false, "replace existing archive with same name")
}

func (c *BackupCommand) Init(args []string) error {
	filename, err := cmd.ZeroOrOneArgs(args)
	if err != nil {
		return err
	}
	c.Filename = filename
	c.VerifyHash = !c.noverify
	return nil
}

func (c *BackupCommand) runBackup(ctx *cmd.Context, client backupClient) error {
	filename, hash, expectedHash, err := client.Backup(c.Filename, !c.Overwrite)
	if err != nil {
		if params.IsCodeNotImplemented(err) {
			return fmt.Errorf("server version not compatible with client version")
		} else {
			return err
		}
	}

	fmt.Fprintln(ctx.Stdout, filename)

	if c.VerifyHash {
		if hash != expectedHash {
			return fmt.Errorf("SHA-1 checksum mismatch (archive still saved)")
		}
	}

	return nil
}

func (c *BackupCommand) Run(ctx *cmd.Context) error {
	client, err := juju.NewAPIClientFromName(c.ConnectionName())
	if err != nil {
		return err
	}
	defer client.Close()

	return c.runBackup(ctx, client)
}
