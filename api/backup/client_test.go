// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup_test

import (
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api/backup"
)

type backupSuite struct {
	baseSuite
}

var _ = gc.Suite(&backupSuite{})

func (s *backupSuite) TestClient(c *gc.C) {
	facade := backup.ExposeFacade(s.client)

	c.Check(facade.Name(), gc.Equals, "Backup")
}
