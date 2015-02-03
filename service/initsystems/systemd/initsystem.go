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
	fops    fileOperations
}

// NewInitSystem returns a new value that implements
// initsystems.InitSystem for Windows.
func NewInitSystem(name string) initsystems.InitSystem {
	return &systemd{
		name:    name,
		newConn: newConn,
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

	statusCh := make(chan string)
	_, err = conn.StartUnit(name, "fail", statusCh)
	if err != nil {
		return errors.Trace(err)
	}

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

	statusCh := make(chan string)
	_, err = conn.StopUnit(name, "fail", statusCh)
	if err != nil {
		return errors.Trace(err)
	}

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
func (is *systemd) Info(name string) (*initsystems.ServiceInfo, error) {
	conn, err := is.newConn()
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer conn.Close()

	units, err := conn.ListUnits()
	if err != nil {
		return nil, errors.Trace(err)
	}

	for _, unit := range units {
		if unit.Name == name {
			return newInfo(unit), nil
		}
	}

	return nil, errors.NotFoundf("service %q", name)
}

// Conf implements service/initsystems.InitSystem.
func (is *systemd) Conf(name string) (*initsystems.Conf, error) {
	if err := initsystems.EnsureStatus(is, name, initsystems.StatusEnabled); err != nil {
		return nil, errors.Trace(err)
	}

	// TODO(ericsnow) Finish!
	return nil, nil
}

// Validate implements service/initsystems.InitSystem.
func (is *systemd) Validate(name string, conf initsystems.Conf) error {
	err := Validate(name, conf)
	return errors.Trace(err)
}

// Serialize implements service/initsystems.InitSystem.
func (is *systemd) Serialize(name string, conf initsystems.Conf) ([]byte, error) {
	data, err := Serialize(name, conf)
	return data, errors.Trace(err)
}

// Deserialize implements service/initsystems.InitSystem.
func (is *systemd) Deserialize(data []byte, name string) (*initsystems.Conf, error) {
	conf, err := Deserialize(data, name)
	return conf, errors.Trace(err)
}
