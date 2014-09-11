// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import (
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/cmd/juju/backups"
	cmdtesting "github.com/juju/juju/cmd/testing"
	"github.com/juju/juju/testing"
)

type BackupCommandSuite struct {
	testing.FakeJujuHomeSuite
}

var _ = gc.Suite(&BackupCommandSuite{})

func (s *BackupCommandSuite) TestBackupsHelp(c *gc.C) {
	cmd := backups.NewBackupsCommand()

	cmdtesting.CheckHelp(c, "juju", cmd)
}
