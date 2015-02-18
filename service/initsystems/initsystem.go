// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// TODO(ericsnow) At some point we should consider moving the
// initsystems package and portions of the service package into another
// repo (e.g. utils).

package initsystems

// InitSystem represents the functionality provided by an init system.
// It encompasses all init services on the host, rather than just juju-
// managed ones.
type InitSystem interface {
	ConfHandler

	// Name returns the init system's name.
	Name() string

	// List gathers the names of all enabled services in the init system
	// and returns them. If any names are passed as arguments then the
	// result will be limited to those names. Otherwise all known
	// service names are returned.
	List(include ...string) ([]string, error)

	// Start causes the named service to be started. If it is already
	// started then errors.AlreadyExists is returned. If the service has
	// not been enabled then errors.NotFound is returned.
	Start(name string) error

	// Stop causes the named service to be stopped. If it is already
	// stopped then errors.NotFound is returned. If the service has
	// not been enabled then errors.NotFound is returned.
	Stop(name string) error

	// Enable adds a new service to the init system with the given name.
	// The conf file at the provided filename is used for the new
	// service. If a service with that name is already enabled then
	// errors.AlreadyExists is returned. The file will be deserialized
	// and validated before the service is enabled.
	Enable(name, filename string) error

	// Disable removes the named service from the init system. If it is
	// not already enabled then errors.NotFound is returned.
	Disable(name string) error

	// IsEnabled determines whether or not the named service is enabled.
	IsEnabled(name string) (bool, error)

	// Check determines whether or not the named service is enabled
	// and matches the conf at the filename.
	Check(name, filename string) (bool, error)

	// Info gathers information about the named service and returns it.
	// If the service is not enabled then errors.NotFound is returned.
	Info(name string) (ServiceInfo, error)

	// Conf composes a Conf for the named service and returns it.
	// If the service is not enabled then errors.NotFound is returned.
	Conf(name string) (Conf, error)
}
