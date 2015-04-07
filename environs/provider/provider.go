// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider

import (
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/storage"
	"github.com/juju/juju/state"
)

type ProviderFactory interface {
	// PrepareForBootstrap prepares an environment for use. Any additional
	// configuration attributes in the returned environment should
	// be saved to be used later. If the environment is already
	// prepared, this call is equivalent to Open.
	PrepareForBootstrap(ctx environs.BootstrapContext, cfg *config.Config) (Provider, error)

	// Open opens the environment and returns it.
	// The configuration must have come from a previously
	// prepared environment.
	Open(cfg *config.Config) (Provider, error)

	// EnvironCapability implements state.Policy.
	EnvironCapability(*config.Config) (state.EnvironCapability, error)
}

// An Environ represents a juju environment as specified
// in the environments.yaml file.
//
// Due to the limitations of some providers (for example ec2), the
// results of the Environ methods may not be fully sequentially
// consistent. In particular, while a provider may retry when it
// gets an error for an operation, it will not retry when
// an operation succeeds, even if that success is not
// consistent with a previous operation.
//
// Even though Juju takes care not to share an Environ between concurrent
// workers, it does allow concurrent method calls into the provider
// implementation.  The typical provider implementation needs locking to
// avoid undefined behaviour when the configuration changes.
type Provider interface {
	Bootstrapper() Bootstrapper
	ComputeBroker(kind string) (ComputeBroker, err)
	NetworkBroker(kind string) (NetworkBroker, err)
	StorageBroker(kind string) (StorageBroker, err)
	GlobalFirewallBroker(kind string) (GlobalFirewallBroker, err)
	InstanceFirewallBroker(kind string) (InstanceFirewallBroker, err)

	// Destroy shuts down all known machines and destroys the
	// rest of the environment. Note that on some providers,
	// very recently started instances may not be destroyed
	// because they are not yet visible.
	//
	// When Destroy has been called, any Environ referring to the
	// same remote environment may become invalid
	Destroy() error

	// Configurer is the factory used to configure this provider.
	Configurer() (Configurer, error)
}
