// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"github.com/juju/errors"
	"github.com/juju/testing"
	"gopkg.in/juju/charm.v5"

	"github.com/juju/juju/process"
	"github.com/juju/juju/process/context"
	jujuctesting "github.com/juju/juju/worker/uniter/runner/jujuc/testing"
)

type StubHookContext struct {
	*jujuctesting.Context
}

func (c StubHookContext) Component(name string) (context.Component, error) {
	found, err := c.Context.Component(name)
	if err != nil {
		return nil, errors.Trace(err)
	}
	compCtx, ok := found.(context.Component)
	if !ok && found != nil {
		return nil, errors.Errorf("wrong component context type registered: %T", found)
	}
	return compCtx, nil
}

type StubContextComponentData struct {
	Procs       map[string]process.Info
	Definitions map[string]charm.Process
	Plugin      process.Plugin
}

type StubContextComponent struct {
	Stub *testing.Stub
	Data StubContextComponentData
}

func NewStubContextComponent(Stub *testing.Stub) *StubContextComponent {
	return &StubContextComponent{
		Stub: Stub,
		Data: StubContextComponentData{
			Procs:       make(map[string]process.Info),
			Definitions: make(map[string]charm.Process),
		},
	}
}

func (c *StubContextComponent) Plugin(info *process.Info) (process.Plugin, error) {
	c.Stub.AddCall("Plugin", info)
	if err := c.Stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	if c.Data.Plugin == nil {
		return &StubPlugin{Stub: c.Stub}, nil
	}
	return c.Data.Plugin, nil
}

func (c *StubContextComponent) Get(id string) (*process.Info, error) {
	c.Stub.AddCall("Get", id)
	if err := c.Stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	info, ok := c.Data.Procs[id]
	if !ok {
		return nil, errors.NotFoundf(id)
	}
	return &info, nil
}

func (c *StubContextComponent) List() ([]string, error) {
	c.Stub.AddCall("List")
	if err := c.Stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	var ids []string
	for k := range c.Data.Procs {
		ids = append(ids, k)
	}
	return ids, nil
}

func (c *StubContextComponent) Set(info process.Info) error {
	c.Stub.AddCall("Set", info)
	if err := c.Stub.NextErr(); err != nil {
		return errors.Trace(err)
	}

	c.Data.Procs[info.ID()] = info
	return nil
}

func (c *StubContextComponent) ListDefinitions() ([]charm.Process, error) {
	c.Stub.AddCall("ListDefinitions")
	if err := c.Stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	var definitions []charm.Process
	for _, definition := range c.Data.Definitions {
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func (c *StubContextComponent) Flush() error {
	c.Stub.AddCall("Flush")
	if err := c.Stub.NextErr(); err != nil {
		return errors.Trace(err)
	}

	return nil
}
