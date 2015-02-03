// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd

import (
	"github.com/juju/juju/service/initsystems"
)

func NewSystemd(fake dbusApi) initsystems.InitSystem {
	return &systemd{
		name:    "systemd",
		newConn: func() (dbusApi, error) { return fake, nil },
	}
}
