// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd

import (
	"github.com/coreos/go-systemd/dbus"
	"github.com/juju/utils/fs"

	"github.com/juju/juju/service/initsystems"
)

type dbusApi interface {
	Close()
	ListUnits() ([]dbus.UnitStatus, error)
	StartUnit(name string, mode string, ch chan<- string) (int, error)
	StopUnit(name string, mode string, ch chan<- string) (int, error)
	EnableUnitFiles(files []string, runtime bool, force bool) (bool, []dbus.EnableUnitFileChange, error)
	DisableUnitFiles(files []string, runtime bool) ([]dbus.DisableUnitFileChange, error)
}

func newConn() (dbusApi, error) {
	return dbus.New()
}

type fileOperations interface {
	ReadFile(filename string) ([]byte, error)
}

func newFileOperations() fileOperations {
	return &fs.Ops{}
}

func newInfo(unit dbus.UnitStatus) initsystems.ServiceInfo {
	status := initsystems.StatusError
	if unit.LoadState == "loaded" {
		var ok bool
		status, ok = statusMap[unit.ActiveState]
		if !ok {
			status = initsystems.StatusStopped
		}
	}

	return initsystems.ServiceInfo{
		Name:        unit.Name,
		Description: unit.Description,
		Status:      status,
	}
}

// statusMap maps DBUS statuses to our internal initsystem status types.
// See: http://www.freedesktop.org/wiki/Software/systemd/dbus/ (Properties: ActiveState)
var statusMap = map[string]string{
	"active":       initsystems.StatusRunning,
	"reloading":    initsystems.StatusStarting,
	"inactive":     initsystems.StatusStopped,
	"failed":       initsystems.StatusStopped,
	"activating":   initsystems.StatusStarting,
	"deactivating": initsystems.StatusStopping,
}
