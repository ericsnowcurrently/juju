// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package procmanager

// Plugin represents the functionality used by Juju of a process
// management plugin.
type Plugin interface {
	// TODO(wwitzel3) determine specific details for storage and networking
	// Launch launches a new process with the given info.
	Launch(image, desc, storage, networking, args string) (ProcessDetails, error)

	// Destroy destroys an existing process.
	Destroy(id string) error

	// Info retrieves information about an existing process, including status.
	Info(id string) (ProcessDetails, error)
}

// ProcessDetails holds information about an existing process as provided by
// the plugin.
type ProcessDetails struct {
	// UniqueID is provided by the plugin as a guaranteed way
	// to identify the process to the plugin.
	UniqueID string

	// Status is the status of the process as reported by the plugin.
	Status string
}

// TODO(wwitzel3) determine storageInfo based on spec/plugin needs
type storageInfo struct{}

// TODO(wwitzel3) determine networkInfo based on spec/plugin needs
type networkInfo struct{}

// PluginResource exposes an API for plugins to call
// to validate and allocate Juju resources.
type PluginResource interface {
	// Storage returns information needed by a plugin about the given
	// storage resource.
	// $ storage-info -> API.PluginResource.Storage
	Storage(storageID string) (storageInfo, error)

	// Networking returns information needed by a plugin about the given
	// networking resource.
	// $ network-info -> API.PluginResource.Networking
	Networking(networking string) (networkInfo, error)
}
