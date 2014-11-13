// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"fmt"
	"io"

	"github.com/juju/cmd"
	"github.com/juju/errors"

	"github.com/juju/juju/api/backup"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/cmd/envcmd"
)

var backupDoc = `
"juju backup" is used to manage backups of the state of a juju environment.
`

const backupPurpose = "create, manage, and restore backups of juju's state"

// Command is the top-level command wrapping all backups functionality.
type Command struct {
	cmd.SuperCommand
}

// NewCommand returns a new backups super-command.
func NewCommand() cmd.Command {
	backupCmd := Command{
		SuperCommand: *cmd.NewSuperCommand(
			cmd.SuperCommandParams{
				Name:        "backup",
				Doc:         backupDoc,
				UsagePrefix: "juju",
				Purpose:     backupPurpose,
			},
		),
	}
	backupCmd.Register(envcmd.Wrap(&CreateCommand{}))
	backupCmd.Register(envcmd.Wrap(&InfoCommand{}))
	backupCmd.Register(envcmd.Wrap(&ListCommand{}))
	backupCmd.Register(envcmd.Wrap(&DownloadCommand{}))
	backupCmd.Register(envcmd.Wrap(&RemoveCommand{}))
	return &backupCmd
}

// APIClient represents the backups API client functionality used by
// the backup command.
type APIClient interface {
	io.Closer
	// Create sends an RPC request to create a new backup.
	Create(notes string) (*params.BackupMetadataResult, error)
	// Info gets the backup's metadata.
	Info(id string) (*params.BackupMetadataResult, error)
	// List gets all stored metadata.
	List() (*params.BackupListResult, error)
	// Download pulls the backup archive file.
	Download(id string) (io.ReadCloser, error)
	// Remove removes the stored backup.
	Remove(id string) error
}

// CommandBase is the base type for backup sub-commands.
type CommandBase struct {
	envcmd.EnvCommandBase
}

// NewAPIClient returns a client for the backup api endpoint.
func (c *CommandBase) NewAPIClient() (APIClient, error) {
	return newAPIClient(c)
}

var newAPIClient = func(c *CommandBase) (APIClient, error) {
	root, err := c.NewAPIRoot()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return backup.NewClient(root), nil
}

// dumpMetadata writes the formatted backup metadata to stdout.
func (c *CommandBase) dumpMetadata(ctx *cmd.Context, result *params.BackupMetadataResult) {
	fmt.Fprintf(ctx.Stdout, "backup ID:       %q\n", result.ID)
	fmt.Fprintf(ctx.Stdout, "checksum:        %q\n", result.Checksum)
	fmt.Fprintf(ctx.Stdout, "checksum format: %q\n", result.ChecksumFormat)
	fmt.Fprintf(ctx.Stdout, "size (B):        %d\n", result.Size)
	fmt.Fprintf(ctx.Stdout, "stored:          %v\n", result.Stored)

	fmt.Fprintf(ctx.Stdout, "started:         %v\n", result.Started)
	fmt.Fprintf(ctx.Stdout, "finished:        %v\n", result.Finished)
	fmt.Fprintf(ctx.Stdout, "notes:           %q\n", result.Notes)

	fmt.Fprintf(ctx.Stdout, "environment ID:  %q\n", result.Environment)
	fmt.Fprintf(ctx.Stdout, "machine ID:      %q\n", result.Machine)
	fmt.Fprintf(ctx.Stdout, "created on host: %q\n", result.Hostname)
	fmt.Fprintf(ctx.Stdout, "juju version:    %v\n", result.Version)
}
