// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider

import (
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state"
)

type Configurer interface {
	// RestrictedConfigAttributes are provider specific attributes stored in
	// the config that really cannot or should not be changed across
	// environments running inside a single juju server.
	RestrictedConfigAttributes() []string

	// PrepareForCreateEnvironment prepares an environment for creation. Any
	// additional configuration attributes are added to the config passed in
	// and returned.  This allows providers to add additional required config
	// for new environments that may be created in an existing juju server.
	PrepareForCreateEnvironment(cfg *config.Config) (*config.Config, error)

	// Validate ensures that config is a valid configuration for this
	// provider, applying changes to it if necessary, and returns the
	// validated configuration.
	// If old is not nil, it holds the previous environment configuration
	// for consideration when validating changes.
	Validate(cfg, old *config.Config) (valid *config.Config, err error)

	// Boilerplate returns a default configuration for the environment in yaml format.
	// The text should be a key followed by some number of attributes:
	//    `environName:
	//        type: environTypeName
	//        attr1: val1
	//    `
	// The text is used as a template (see the template package) with one extra template
	// function available, rand, which expands to a random hexadecimal string when invoked.
	BoilerplateConfig() string

	// SecretAttrs filters the supplied configuration returning only values
	// which are considered sensitive. All of the values of these secret
	// attributes need to be strings.
	SecretAttrs(cfg *config.Config) (map[string]string, error)
}

// ConfigGetter implements access to an environment's configuration.
type ConfigGetter interface {
	// Config returns the configuration data with which the Environ was created.
	// Note that this is not necessarily current; the canonical location
	// for the configuration data is stored in the state.
	Config() *config.Config
}
