// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"github.com/juju/errors"
	"github.com/juju/testing"
	"gopkg.in/juju/charm.v5"

	"github.com/juju/juju/process"
)

type StubPluginData struct {
	Details process.Details
	Status  process.PluginStatus
}

type StubPlugin struct {
	Stub *testing.Stub
	Data StubPluginData
}

func (c *StubPlugin) Launch(definition charm.Process) (process.Details, error) {
	c.Stub.AddCall("Launch", definition)
	if err := c.Stub.NextErr(); err != nil {
		return c.Data.Details, errors.Trace(err)
	}

	return c.Data.Details, nil
}

func (c *StubPlugin) Destroy(id string) error {
	c.Stub.AddCall("Destroy", id)
	if err := c.Stub.NextErr(); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *StubPlugin) Status(id string) (process.PluginStatus, error) {
	c.Stub.AddCall("Status", id)
	if err := c.Stub.NextErr(); err != nil {
		return c.Data.Status, errors.Trace(err)
	}

	return c.Data.Status, nil
}
