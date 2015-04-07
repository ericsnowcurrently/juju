// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider

import (
	"github.com/juju/juju/instance"
	"github.com/juju/juju/network"
)

type NetworkBroker interface {
	// Addresses returns a list of hostnames or ip addresses
	// associated with the instance.
	Addresses(id instance.Id) ([]network.Address, error)

	AllocateAddress(instId instance.Id, netId network.Id) (network.Address, error)
	ListNetworks() ([]network.BasicInfo, error)
}

type GlobalFirewallBroker interface {
	// OpenPorts opens the given port ranges for the whole environment.
	// Must only be used if the environment was setup with the
	// FwGlobal firewall mode.
	OpenPorts(ports []network.PortRange) error

	// ClosePorts closes the given port ranges for the whole environment.
	// Must only be used if the environment was setup with the
	// FwGlobal firewall mode.
	ClosePorts(ports []network.PortRange) error

	// Ports returns the port ranges opened for the whole environment.
	// Must only be used if the environment was setup with the
	// FwGlobal firewall mode.
	Ports() ([]network.PortRange, error)
}

type InstanceFirewallBroker interface {
	// OpenPorts opens the given port ranges on the instance, which
	// should have been started with the given machine id.
	OpenPorts(instId instance.Id, machineId string, ports []network.Port) error

	// ClosePorts closes the given port ranges on the instance, which
	// should have been started with the given machine id.
	ClosePorts(instId instance.Id, machineId string, ports []network.Port) error

	// Ports returns the set of port ranges open on the instance,
	// which should have been started with the given machine id. The
	// port ranges are returned as sorted by network.SortPortRanges().
	Ports(instId instance.Id, machineId string) ([]network.Port, error)
}
