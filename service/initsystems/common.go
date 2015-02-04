// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package initsystems

import (
	"github.com/juju/errors"
)

type enabledChecker interface {
	IsEnabled(name string) (bool, error)
}

// EnsureEnabled may be used by InitSystem implementations to ensure
// that the named service has been enabled. This is important for
// operations where the service must first be enabled.
func EnsureEnabled(name string, is enabledChecker) error {
	enabled, err := is.IsEnabled(name)
	if err != nil {
		return errors.Trace(err)
	}
	if !enabled {
		return errors.NotFoundf("service %q", name)
	}
	return nil
}

// FilterNames filters out any name in names that isn't in include.
func FilterNames(names, include []string) []string {
	if len(include) == 0 {
		return names
	}

	var filtered []string
	for _, name := range names {
		for _, included := range include {
			if name == included {
				filtered = append(filtered, name)
				break
			}
		}
	}
	return filtered
}

type deserializer interface {
	Deserialize(data []byte) (*Conf, error)
}

type fileOperations interface {
	ReadFile(filename string) ([]byte, error)
}

// ReadConf wraps the operations of reading the file and deserializing
// it (and validating the resulting conf).
func ReadConf(name, filename string, is deserializer, fops fileOperations) (*Conf, error) {
	data, err := fops.ReadFile(filename)
	if err != nil {
		return nil, errors.Trace(err)
	}
	conf, err := is.Deserialize(data)
	return conf, errors.Trace(err)
}
