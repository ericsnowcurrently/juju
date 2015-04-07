// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider

import (
	"io"
	"os"

	"github.com/juju/juju/constraints"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/cloudinit"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/storage"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/network"
	"github.com/juju/juju/storage"
	"github.com/juju/juju/tools"
)

// TODO(ericsnow) Add a ZoneBroker that exposes:
// InstanceDistributor(*config.Config) (InstanceDistributor, error)

type ComputeBroker interface {
	// ConstraintsValidator implements state.Policy.
	ConstraintsValidator() (constraints.Validator, error)

	// Prechecker implements state.Policy.
	Prechecker(*config.Config) (state.Prechecker, error)

	// StartInstance asks for a new instance to be created, associated with
	// the provided config in machineConfig. The given config describes the juju
	// state for the new instance to connect to. The config MachineNonce, which must be
	// unique within an environment, is used by juju to protect against the
	// consequences of multiple instances being started with the same machine
	// id.
	StartInstance(args StartInstanceParams) (*StartInstanceResult, error)

	// StopInstances shuts down the instances with the specified IDs.
	// Unknown instance IDs are ignored, to enable idempotency.
	StopInstances(...instance.Id) error

	// AllInstances returns all instances currently known to the broker.
	AllInstances() ([]instance.Id, error)

	// Instances returns a slice of instances corresponding to the
	// given instance ids.  If no instances were found, but there
	// was no other error, it will return ErrNoInstances.  If
	// some but not all the instances were found, the returned slice
	// will have some nil slots, and an ErrPartialInstances error
	// will be returned.
	Instances(ids []instance.Id) ([]instance.Instance, error)

	// StateServerInstances returns the IDs of instances corresponding
	// to Juju state servers. If there are no state server instances,
	// ErrNoInstances is returned. If it can be determined that the
	// environment has not been bootstrapped, then ErrNotBootstrapped
	// should be returned instead.
	StateServerInstances() ([]instance.Id, error)

	// Status returns the provider-specific status for the instance.
	Status(id instance.Id) (string, error)
}

// StartInstanceParams holds parameters for the
// InstanceBroker.StartInstance method.
type StartInstanceParams struct {
	// Constraints is a set of constraints on
	// the kind of instance to create.
	Constraints constraints.Value

	// Tools is a list of tools that may be used
	// to start a Juju agent on the machine.
	Tools tools.List

	// MachineConfig describes the machine's configuration.
	MachineConfig *cloudinit.MachineConfig

	// Placement, if non-empty, contains an environment-specific
	// placement directive that may be used to decide how the
	// instance should be started.
	Placement string

	// DistributionGroup, if non-nil, is a function
	// that returns a slice of instance.Ids that belong
	// to the same distribution group as the machine
	// being provisioned. The InstanceBroker may use
	// this information to distribute instances for
	// high availability.
	DistributionGroup func() ([]instance.Id, error)

	// Volumes is a set of parameters for volumes that should be created.
	//
	// StartInstance need not check the value of the Attachment field,
	// as it is guaranteed that any volumes in this list are designated
	// for attachment to the instance being started.
	Volumes []storage.VolumeParams

	// NetworkInfo is an optional list of network interface details,
	// necessary to configure on the instance.
	NetworkInfo []network.InterfaceInfo
}

// StartInstanceResult holds the result of an
// InstanceBroker.StartInstance method call.
type StartInstanceResult struct {
	// ID identifies the instance uniquely (for the provider).
	ID instance.Id

	// HardwareCharacteristics represents the hardware characteristics
	// of the newly created instance.
	Hardware *instance.HardwareCharacteristics

	// NetworkInfo contains information about how to configure network
	// interfaces on the instance. Depending on the provider, this
	// might be the same StartInstanceParams.NetworkInfo or may be
	// modified as needed.
	NetworkInfo []network.InterfaceInfo

	// Volumes contains a list of volumes created, each one having the
	// same Name as one of the VolumeParams in StartInstanceParams.Volumes.
	// VolumeAttachment information is reported separately.
	Volumes []storage.Volume

	// VolumeAttachments contains a attachment-specific information about
	// volumes that were attached to the started instance.
	VolumeAttachments []storage.VolumeAttachment
}
