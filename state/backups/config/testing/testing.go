// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/backups/config"
	"github.com/juju/juju/testing"
	"github.com/juju/juju/utils"
)

func NewPaths(c *gc.C) (utils.Workspace, config.Paths) {
	ws := testing.NewTestWorkspace(c)

	paths := config.ReRoot(&config.DefaultPaths, ws.RootDir())

	// Prep the fake FS.
	dir, _ := ws.Sub("/var/lib/juju")
	dir.Touch("tools")
	dir.Touch("system-identity")
	dir.Touch("nonce.txt")
	dir.Touch("server.pem")
	dir.Touch("shared-secret")

	dir, _ = ws.Sub("/var/lib/juju/agents")
	dir.Touch("machine-0.conf")

	dir, _ = ws.Sub("/var/log/juju")
	dir.Touch("all-machines.log")
	dir.Touch("machine-0.log")

	dir, _ = ws.Sub("/etc/init")
	dir.Touch("jujud-machine-0.conf")
	dir.Touch("juju-db.conf")

	dir, _ = ws.Sub("/etc/rsyslog.d")
	dir.Touch("spam-juju.conf")

	dir, _ = ws.Sub("/home/ubuntu/.ssh")
	dir.Touch("authorized_keys")

	return ws, paths
}

func NewDBInfo(c *gc.C) config.DBInfo {
	dbConnInfo := config.NewDBConnInfo(
		"some-host.com:8080",
		"awesome",
		"bad-pw",
	)
	dbInfo, err := config.NewDBInfo(dbConnInfo)
	c.Assert(err, gc.IsNil)
	return dbInfo
}

func NewConfig(c *gc.C) (utils.Workspace, config.BackupsConfig) {
	ws, paths := NewPaths(c)
	dbInfo := NewDBInfo(c)

	conf, err := config.NewBackupsConfig(dbInfo, paths)
	c.Assert(err, gc.IsNil)

	return ws, conf
}
