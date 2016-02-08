// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmcmd

import (
	"github.com/juju/cmd"

	//corecmd "github.com/juju/juju/cmd"
)

var charmDoc = `
"juju charm" is the the juju CLI equivalent of the "charm" command used
by charm authors, though only applicable functionality is mirrored.
`

const charmPurpose = "interact with charms"

// Command is the top-level command wrapping all backups functionality.
type Command struct {
	*cmd.SuperCommand

	//apiCtx   *corecmd.APIContext
}

// NewSuperCommand returns a new charm super-command.
func NewSuperCommand() (*Command, error) {
	//apiCtx, err := corecmd.NewAPIContext()
	//if err != nil {
	//	return errors.Trace(err)
	//}

	//httpClient, err := apiCtx.HTTPClient()
	//csClient := newCharmStoreClient(httpClient)

	charmCmd := &Command{
		// TODO(ericsnow) Use juju/juju/cmd.NewSuperCommand()?
		SuperCommand: cmd.NewSuperCommand(
			cmd.SuperCommandParams{
				Name:        "charm",
				Doc:         charmDoc,
				UsagePrefix: "juju",
				Purpose:     charmPurpose,
			},
		),
		//apiCtx: apiCtx,
	}
	//charmCmd.Register(newXXXCommand())
	return charmCmd, nil
}

// Run implements cmd.Command.
func (c *Command) Run(ctx *cmd.Context) error {
	//defer c.apiCtx.Close()
	return c.SuperCommand.Run(ctx)
}
