// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider

import (
	"github.com/juju/juju/environs/storage"
)

// TODO(ericsnow) Eliminate storage.

// StorageBroker implements storage access for an environment.
type StorageBroker interface {
	// Storage returns storage specific to the environment.
	Storage() storage.Storage
}
