// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package plugin contains the code that interfaces with plugins for workload
// technologies such as Docker, Rocket, or systemd.
package plugin

import (
	"github.com/juju/errors"

	"github.com/juju/juju/workload"
)

var builtinPlugins = map[string]workload.Plugin{
	"docker": NewDockerPlugin(nil),
}

// Find returns the plugin for the given name.
func Find(name string) (workload.Plugin, error) {
	plugin, ok := builtinPlugins[name]
	if !ok {
		return nil, errors.NotFoundf("plugin %q", name)
	}
	return plugin, nil
}
