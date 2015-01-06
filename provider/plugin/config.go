// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package plugin

import (
	"github.com/juju/errors"
	"github.com/juju/schema"

	"github.com/juju/juju/environs/config"
)

type environConfig struct {
	*config.Config
	attrs map[string]interface{}
}

/*
func (c *environConfig) auth() gceAuth {
	return gceAuth{
		clientID:    c.attrs[cfgClientID].(string),
		clientEmail: c.attrs[cfgClientEmail].(string),
		privateKey:  []byte(c.attrs[cfgPrivateKey].(string)),
	}
}

func (c *environConfig) newConnection() *gceConnection {
	return &gceConnection{
		region:    c.attrs[cfgRegion].(string),
		projectID: c.attrs[cfgProjectID].(string),
	}
}
*/

func validateAndUpdateConfig(cfg, old *config.Config) (map[string]interface{}, error) {
	// Check for valid changes and coerce the values (base config first
	// then custom).
	if err := config.Validate(cfg, old); err != nil {
		return nil, errors.Trace(err)
	}
	validated, err := cfg.ValidateUnknownAttrs(configFields, configDefaults)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// TODO(ericsnow) Pull missing values from shell environment variables.

	// Check that no immutable fields have changed.
	if old != nil {
		attrs := old.UnknownAttrs()
		for _, field := range configImmutableFields {
			if attrs[field] != validated[field] {
				return nil, errors.Errorf("%s: cannot change from %v to %v", field, attrs[field], validated[field])
			}
		}
	}

	return validated, nil
}

func validateAndUpdateConfig(cfg, old *config.Config) (*environConfig, error) {
	return &environConfig{cfg, validated}, nil
}

func validateAndUpdateConfig(cfg, old *config.Config) (*environConfig, error) {
	// Check sanity of config fields.
	if err := validateConfigHook(ecfg); err != nil {
		return nil, errors.Trace(handleInvalidField(err))
	}

	// TODO(ericsnow) Follow up with someone on if it is appropriate
	// to call Apply here.
	cfg, err = ecfg.Config.Apply(ecfg.attrs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ecfg.Config = cfg
	return ecfg, nil
}

func handleInvalidField(err error) error {
	vErr := err.(*config.InvalidConfigValue)
	if vErr.Reason == nil && vErr.Value == "" {
		key := osEnvFields[vErr.Key]
		return errors.Errorf("%s: must not be empty", key)
	}
	return err
}
