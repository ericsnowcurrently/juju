// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"fmt"

	"github.com/juju/errors"
	"launchpad.net/gnuflag"

	"github.com/juju/juju/state/api/params"
)

const createDoc = `
Request that juju create a backup of its state and print the backup's
unique ID.  You may provide a note to associate with the backup.

The backup archive and associated metadata are stored in juju and
will be lost when the environment is destroyed.
`

type BackupsCreateCommand struct {
	BackupsCommandBase
	Quiet bool
	Notes string
}

func (c *BackupsCreateCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "create",
		Args:    "[--quiet] [<notes>]",
		Purpose: "create a backup",
		Doc:     createDoc,
	}
}

func (c *BackupsCreateCommand) SetFlags(f *gnuflag.FlagSet) {
	f.BoolVar(&c.Quiet, "quiet", false, "do not print the metadata")
}

func (c *BackupsCreateCommand) Init(args []string) error {
	notes, err := cmd.ZeroOrOneArgs(args)
	if err != nil {
		return err
	}
	c.Notes = notes
	return nil
}

func (c *BackupsCreateCommand) Run(ctx *cmd.Context) error {
	client, err := c.client()
	if err != nil {
		return errors.Trace(err)
	}
	defer client.Close()

	result, err := client.Create(c.Notes)
	if params.IsCodeNotImplemented(failure) {
		return errors.New("server version not compatible with client version")
	} else if err != nil {
		return errors.Trace(err)
	}

	if !c.Quiet {
		c.dumpMetadata(ctx, result)
	}

	fmt.Fprintln(ctx.Stdout, result.ID)
	return nil
}
