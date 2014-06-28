// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client

import (
	"github.com/juju/charm"

	"github.com/juju/juju/instance"
	"github.com/juju/juju/network"
	"github.com/juju/juju/state/api/params"
)

// NetworksSpecification holds the enabled and disabled networks for a
// service.
type NetworksSpecification struct {
	Enabled  []string
	Disabled []string
}

// AgentStatus holds status info about a machine or unit agent.
type AgentStatus struct {
	Status  params.Status
	Info    string
	Data    params.StatusData
	Version string
	Life    string
	Err     error
}

// MachineStatus holds status info about a machine.
type MachineStatus struct {
	Agent AgentStatus

	// The following fields mirror fields in AgentStatus (introduced
	// in 1.19.x). The old fields below are being kept for
	// compatibility with old clients. They can be removed once API
	// versioning lands or for 1.21, whichever comes first.
	AgentState     params.Status
	AgentStateInfo string
	AgentVersion   string
	Life           string
	Err            error

	DNSName       string
	InstanceId    instance.Id
	InstanceState string
	Series        string
	Id            string
	Containers    map[string]MachineStatus
	Hardware      string
	Jobs          []params.MachineJob
	HasVote       bool
	WantsVote     bool
}

// ServiceStatus holds status info about a service.
type ServiceStatus struct {
	Err           error
	Charm         string
	Exposed       bool
	Life          string
	Relations     map[string][]string
	Networks      NetworksSpecification
	CanUpgradeTo  string
	SubordinateTo []string
	Units         map[string]UnitStatus
}

// UnitStatus holds status info about a unit.
type UnitStatus struct {
	Agent AgentStatus

	// See the comment in MachineStatus regarding these fields.
	AgentState     params.Status
	AgentStateInfo string
	AgentVersion   string
	Life           string
	Err            error

	Machine       string
	OpenedPorts   []string
	PublicAddress string
	Charm         string
	Subordinates  map[string]UnitStatus
}

// RelationStatus holds status info about a relation.
type RelationStatus struct {
	Id        int
	Key       string
	Interface string
	Scope     charm.RelationScope
	Endpoints []EndpointStatus
}

// EndpointStatus holds status info about a single endpoint
type EndpointStatus struct {
	ServiceName string
	Name        string
	Role        charm.RelationRole
	Subordinate bool
}

func (epStatus *EndpointStatus) String() string {
	return epStatus.ServiceName + ":" + epStatus.Name
}

// NetworkStatus holds status info about a network.
type NetworkStatus struct {
	Err        error
	ProviderId network.Id
	CIDR       string
	VLANTag    int
}

// Status holds information about the status of a juju environment.
type Status struct {
	EnvironmentName string
	Machines        map[string]MachineStatus
	Services        map[string]ServiceStatus
	Networks        map[string]NetworkStatus
	Relations       []RelationStatus
}

// Status returns the status of the juju environment.
func (c *Client) Status(patterns []string) (*Status, error) {
	var result Status
	p := params.StatusParams{Patterns: patterns}
	if err := c.call("FullStatus", p, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// LegacyMachineStatus holds just the instance-id of a machine.
type LegacyMachineStatus struct {
	InstanceId string // Not type instance.Id just to match original api.
}

// LegacyStatus holds minimal information on the status of a juju environment.
type LegacyStatus struct {
	Machines map[string]LegacyMachineStatus
}

// LegacyStatus is a stub version of Status that 1.16 introduced. Should be
// removed along with structs when api versioning makes it safe to do so.
func (c *Client) LegacyStatus() (*LegacyStatus, error) {
	var result LegacyStatus
	if err := c.call("Status", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
