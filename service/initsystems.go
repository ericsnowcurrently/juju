// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"io/ioutil"
	"strings"

	"github.com/juju/errors"

	"github.com/juju/juju/service/initsystems"
	"github.com/juju/juju/service/initsystems/upstart"
	"github.com/juju/juju/service/initsystems/windows"
	"github.com/juju/juju/version"
)

// These are the names of the juju-compatible init systems.
const (
	InitSystemWindows = "windows"
	InitSystemUpstart = "upstart"
	//InitSystemSystemd = "systemd"
)

var (
	// linuxInitNames maps the executable from PID 1 onto the name of
	// an init system.
	linuxInitNames = map[string]string{
		"/sbin/init": InitSystemUpstart,
		//"/sbin/systemd": InitSystemSystemd,
	}
)

// newInitSystem returns an InitSystem implementation based on the
// provided name. If the name is unrecognized then errors.NotFound is
// returned.
func newInitSystem(name string) (initsystems.InitSystem, error) {
	switch name {
	case InitSystemWindows:
		return windows.NewInitSystem(name), nil
	case InitSystemUpstart:
		return upstart.NewInitSystem(name), nil
	}
	return nil, errors.NotFoundf("init system implementation for %q", name)
}

// TODO(ericsnow) Support discovering init system on remote host.

// discoverInitSystem determines which init system is running and
// returns its name.
func discoverInitSystem() (string, error) {
	if version.Current.OS == version.Windows {
		return InitSystemWindows, nil
	}

	executable, err := findInitExecutable()
	if err != nil {
		return "", errors.Annotate(err, "while finding init exe")
	}

	name, ok := linuxInitNames[executable]
	if !ok {
		return "", errors.New("unrecognized init system")
	}

	return name, nil
}

var findInitExecutable = func() (string, error) {
	data, err := ioutil.ReadFile("/proc/1/cmdline")
	if err != nil {
		return "", errors.Trace(err)
	}
	return strings.Fields(string(data))[0], nil
}
