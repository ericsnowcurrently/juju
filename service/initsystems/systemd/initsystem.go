// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd

import (
	"github.com/juju/errors"

	"github.com/juju/juju/service/initsystems"
)

const (
	// Name is the name of the init system.
	Name = "systemd"
)

func init() {
	initsystems.Register(Name, initsystems.InitSystemDefinition{
		Name:        Name,
		OSNames:     []string{"!windows"},
		Executables: []string{"/sbin/systemd"},
		New:         NewInitSystem,
	})
}

type systemd struct {
	name    string
	newConn func() (dbusApi, error)
	newChan func() chan string
	fops    fileOperations
}

// NewInitSystem returns a new value that implements
// initsystems.InitSystem for Windows.
func NewInitSystem(name string) initsystems.InitSystem {
	return &systemd{
		name:    name,
		newConn: newConn,
		newChan: func() chan string { return make(chan string) },
		fops:    newFileOperations(),
	}
}

// Name implements service/initsystems.InitSystem.
func (is *systemd) Name() string {
	return is.name
}

// List implements service/initsystems.InitSystem.
func (is *systemd) List(include ...string) ([]string, error) {
	conn, err := is.newConn()
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer conn.Close()

	units, err := conn.ListUnits()
	if err != nil {
		return nil, errors.Trace(err)
	}

	var services []string
	for _, unit := range units {
		services = append(services, unit.Name)
	}

	return initsystems.FilterNames(services, include), nil
}

// Start implements service/initsystems.InitSystem.
func (is *systemd) Start(name string) error {
	if err := initsystems.EnsureStatus(is, name, initsystems.StatusStopped); err != nil {
		return errors.Trace(err)
	}

	conn, err := is.newConn()
	if err != nil {
		return errors.Trace(err)
	}
	defer conn.Close()

	statusCh := is.newChan()
	_, err = conn.StartUnit(name, "fail", statusCh)
	if err != nil {
		return errors.Trace(err)
	}

	// TODO(ericsnow) Add timeout support?
	status := <-statusCh
	if status != "done" {
		return errors.Errorf("failed to start service %s", name)
	}

	return nil
}

// Stop implements service/initsystems.InitSystem.
func (is *systemd) Stop(name string) error {
	if err := initsystems.EnsureStatus(is, name, initsystems.StatusRunning); err != nil {
		return errors.Trace(err)
	}

	conn, err := is.newConn()
	if err != nil {
		return errors.Trace(err)
	}
	defer conn.Close()

	statusCh := is.newChan()
	_, err = conn.StopUnit(name, "fail", statusCh)
	if err != nil {
		return errors.Trace(err)
	}

	// TODO(ericsnow) Add timeout support?
	status := <-statusCh
	if status != "done" {
		return errors.Errorf("failed to stop service %s", name)
	}

	return err
}

// Enable implements service/initsystems.InitSystem.
func (is *systemd) Enable(name, filename string) error {
	if err := initsystems.EnsureStatus(is, name, initsystems.StatusDisabled); err != nil {
		return errors.Trace(err)
	}

	conn, err := is.newConn()
	if err != nil {
		return errors.Trace(err)
	}
	defer conn.Close()

	// TODO(ericsnow) The filename may need to have the format of
	// "<unit name>.service.
	// TODO(ericsnow) We may need to use conn.LinkUnitFiles either
	// instead of or in conjunction with EnableUnitFiles.
	_, _, err = conn.EnableUnitFiles([]string{filename}, false, true)

	return errors.Trace(err)
}

// Disable implements service/initsystems.InitSystem.
func (is *systemd) Disable(name string) error {
	if err := initsystems.EnsureStatus(is, name, initsystems.StatusEnabled); err != nil {
		return errors.Trace(err)
	}

	conn, err := is.newConn()
	if err != nil {
		return errors.Trace(err)
	}
	defer conn.Close()

	// TODO(ericsnow) We may need the original file name (or make sure
	// the unit conf is on the systemd search path.
	_, err = conn.DisableUnitFiles([]string{name}, false)

	return errors.Trace(err)
}

// IsEnabled implements service/initsystems.InitSystem.
func (is *systemd) IsEnabled(name string) (bool, error) {
	names, err := is.List(name)
	if err != nil {
		return false, errors.Trace(err)
	}

	return len(names) > 0, nil
}

// Check implements service/initsystems.InitSystem.
func (is *systemd) Check(name string, filename string) (bool, error) {
	matched, err := initsystems.CheckConf(name, filename, is, is.fops)
	return matched, errors.Trace(err)
}

// Info implements service/initsystems.InitSystem.
func (is *systemd) Info(name string) (initsystems.ServiceInfo, error) {
	var info initsystems.ServiceInfo

	conn, err := is.newConn()
	if err != nil {
		return info, errors.Trace(err)
	}
	defer conn.Close()

	units, err := conn.ListUnits()
	if err != nil {
		return info, errors.Trace(err)
	}

	for _, unit := range units {
		if unit.Name == name {
			return newInfo(unit), nil
		}
	}

	return info, errors.NotFoundf("service %q", name)
}

// Conf implements service/initsystems.InitSystem.
func (is *systemd) Conf(name string) (initsystems.Conf, error) {
	var conf initsystems.Conf

	if err := initsystems.EnsureStatus(is, name, initsystems.StatusEnabled); err != nil {
		return conf, errors.Trace(err)
	}

	// TODO(ericsnow) Finish!
	// This may involve conn.GetUnitProperties...
	return conf, nil
}

// Validate implements service/initsystems.InitSystem.
func (is *systemd) Validate(name string, conf initsystems.Conf) (string, error) {
	confName, err := Validate(name, conf)
	return confName, errors.Trace(err)
}

// Serialize implements service/initsystems.InitSystem.
func (is *systemd) Serialize(name string, conf initsystems.Conf) ([]byte, error) {
	data, err := Serialize(name, conf)
	return data, errors.Trace(err)
}

// Deserialize implements service/initsystems.InitSystem.
func (is *systemd) Deserialize(data []byte, name string) (initsystems.Conf, error) {
	conf, err := Deserialize(data, name)
	return conf, errors.Trace(err)
}
