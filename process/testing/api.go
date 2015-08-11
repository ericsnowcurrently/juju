// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"github.com/juju/errors"
	"github.com/juju/testing"
	"gopkg.in/juju/charm.v5"

	"github.com/juju/juju/process"
)

type StubAPIClientData struct {
	Procs       map[string]process.Info
	Definitions map[string]charm.Process
}

func (d *StubAPIClientData) SetNew(ids ...string) []process.Info {
	var procs []process.Info
	for _, id := range ids {
		name, pluginID := process.ParseID(id)
		if name == "" {
			panic("missing name")
		}
		if pluginID == "" {
			panic("missing id")
		}
		proc := process.Info{
			Process: charm.Process{
				Name: name,
				Type: "myplugin",
			},
			Status: process.Status{
				State: process.StateRunning,
			},
			Details: process.Details{
				ID: pluginID,
				Status: process.PluginStatus{
					State: "okay",
				},
			},
		}
		d.Procs[id] = proc
		procs = append(procs, proc)
	}
	return procs
}

type StubAPIClient struct {
	Stub *testing.Stub
	Data StubAPIClientData
}

func NewStubAPIClient(Stub *testing.Stub) *StubAPIClient {
	return &StubAPIClient{
		Stub: Stub,
		Data: StubAPIClientData{
			Procs:       make(map[string]process.Info),
			Definitions: make(map[string]charm.Process),
		},
	}
}

func (c *StubAPIClient) AllDefinitions() ([]charm.Process, error) {
	c.Stub.AddCall("AllDefinitions")
	if err := c.Stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	var definitions []charm.Process
	for _, definition := range c.Data.Definitions {
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func (c *StubAPIClient) ListProcesses(ids ...string) ([]process.Info, error) {
	c.Stub.AddCall("ListProcesses", ids)
	if err := c.Stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	var procs []process.Info
	if ids == nil {
		for _, proc := range c.Data.Procs {
			procs = append(procs, proc)
		}
	} else {
		for _, id := range ids {
			proc, ok := c.Data.Procs[id]
			if !ok {
				return nil, errors.NotFoundf("proc %q", id)
			}
			procs = append(procs, proc)
		}
	}
	return procs, nil
}

func (c *StubAPIClient) RegisterProcesses(procs ...process.Info) ([]string, error) {
	c.Stub.AddCall("RegisterProcesses", procs)
	if err := c.Stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	var ids []string
	for _, proc := range procs {
		c.Data.Procs[proc.Name] = proc
		ids = append(ids, proc.ID())
	}
	return ids, nil
}
