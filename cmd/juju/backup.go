// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"launchpad.net/gnuflag"

	"github.com/juju/cmd"
	"github.com/juju/juju/cmd/envcmd"
	"github.com/juju/juju/juju"
	"github.com/juju/juju/state/api"
	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/state/backup"
)

var (
	createEmptyFile = backup.CreateEmptyFile
)

type backupClient interface {
	Backup(io.Writer) *api.BackupResult
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

func (c *BackupCommand) handleBackupFailure(
	failure *params.Error, filename string,
) error {
	if failure == nil {
		return nil
	}
	if !c.Overwrite {
		logger.Debugf("removing temp file: %s", filename)
		err := os.Remove(filename)
		if err != nil {
			logger.Infof("could not remove temp file (%s): %v", filename, err)
		}
	}
	if params.IsCodeNotImplemented(failure) {
		return fmt.Errorf("server version not compatible with client version")
	} else {
		return failure
	}
}

func (c *BackupCommand) runBackup(ctx *cmd.Context, client backupClient) error {
	// Get an empty backup file ready *before* sending the request.
	excl := !c.Overwrite
	archive, filename, err := createEmptyFile(c.Filename, 0600, excl)
	if err != nil {
		return fmt.Errorf("error while preparing backup file: %v", err)
	}
	defer archive.Close()
	isTemp := (filename != c.Filename)
	absfilename, err := filepath.Abs(filename)
	if err == nil { // Otherwise we stick with the old filename.
		filename = absfilename
	}
	logger.Debugf("prepared empty backup file: %q", filename)

	// Do the backup.
	res := client.Backup(archive)
	err = c.handleBackupFailure(res.Failure(), filename)
	if err != nil {
		return err
	}
	err = nil

	// Compare the checksums.
	if c.VerifyHash && !res.VerifyHash() {
		err = fmt.Errorf("SHA-1 checksum mismatch (archive still saved)")
	} else if isTemp {
		target := filepath.Join(filepath.Base(filename), res.FilenameFromServer())
		err = os.Rename(filename, target)
		if err != nil {
			err = fmt.Errorf("could not move tempfile to new location: %v", err)
		} else {
			logger.Infof("backup file moved to %s", target)
		}
	}

	// Print out the filename.
	fmt.Fprintln(ctx.Stdout, filename)
	return err
}

func (c *BackupCommand) Run(ctx *cmd.Context) error {
	client, err := juju.NewAPIClientFromName(c.ConnectionName())
	if err != nil {
		return err
	}
	defer client.Close()

	return c.runBackup(ctx, client)
}
