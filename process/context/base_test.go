// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package context_test

import (
	"reflect"

	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v5"

	components "github.com/juju/juju/component/all"
	"github.com/juju/juju/process"
	proctesting "github.com/juju/juju/process/testing"
	"github.com/juju/juju/utils"
	jujuctesting "github.com/juju/juju/worker/uniter/runner/jujuc/testing"
)

func init() {
	utils.Must(components.RegisterForServer())
}

type baseSuite struct {
	jujuctesting.ContextSuite
	proc process.Info
}

func (s *baseSuite) SetUpTest(c *gc.C) {
	s.ContextSuite.SetUpTest(c)

	s.proc = s.newProc("proc A", "docker", "", "")
}

func (s *baseSuite) newProc(name, ptype, id, status string) process.Info {
	info := process.Info{
		Process: charm.Process{
			Name: name,
			Type: ptype,
		},
		Details: process.Details{
			ID: id,
			Status: process.PluginStatus{
				State: status,
			},
		},
	}
	if status != "" {
		info.Status = process.Status{
			State: process.StateRunning,
		}
	}
	return info
}

func (s *baseSuite) NewHookContext() (*proctesting.StubHookContext, *jujuctesting.ContextInfo) {
	ctx, info := s.ContextSuite.NewHookContext()
	return &proctesting.StubHookContext{ctx}, info
}

func checkProcs(c *gc.C, procs, expected []process.Info) {
	if !c.Check(procs, gc.HasLen, len(expected)) {
		return
	}
	for _, proc := range procs {
		matched := false
		for _, expProc := range expected {
			if reflect.DeepEqual(proc, expProc) {
				matched = true
				break
			}
		}
		if !matched {
			c.Errorf("%#v != %#v", procs, expected)
			return
		}
	}
}
