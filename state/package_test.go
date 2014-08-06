// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state_test

import (
	stdtesting "testing"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/api/params"
	statetesting "github.com/juju/juju/state/testing"
	"github.com/juju/juju/testing"
)

// TestPackage integrates the tests into gotest.
func TestPackage(t *stdtesting.T) {
	testing.MgoTestPackage(t)
}

// preventUnitDestroyRemove sets a non-pending status on the unit, and hence
// prevents it from being unceremoniously removed from state on Destroy. This
// is useful because several tests go through a unit's lifecycle step by step,
// asserting the behaviour of a given method in each state, and the unit quick-
// remove change caused many of these to fail.
func preventUnitDestroyRemove(c *gc.C, u *state.Unit) {
	err := u.SetStatus(params.StatusStarted, "", nil)
	c.Assert(err, gc.IsNil)
}

//---------------------------
// the common suite

type BaseStateSuite struct {
	testing.BaseMgoSuite
	State  *state.State
	policy statetesting.MockPolicy
}

func (s *BaseStateSuite) SetUpTest(c *gc.C) {
	s.BaseMgoSuite.SetUpTest(c)
	s.policy = statetesting.MockPolicy{}
	s.State = s.initialize(c, nil, &s.policy)
}

func (s *BaseStateSuite) TearDownTest(c *gc.C) {
	if s.State != nil {
		// If setup fails, we don't have a State yet
		s.State.Close()
	} else {
		c.Logf("skipping State.Close() due to previous error")
	}
	s.BaseMgoSuite.SetUpTest(c)
}

// TestingInitialize initializes the state and returns it. If state was not
// already initialized, and cfg is nil, the minimal default environment
// configuration will be used.
func (s *BaseStateSuite) initialize(c *gc.C, cfg *config.Config, policy state.Policy) *state.State {
	if cfg == nil {
		cfg = testing.EnvironConfig(c)
	}
	st, err := state.Initialize(s.MongoInfo(), cfg, s.DialOpts(), policy)
	c.Assert(err, gc.IsNil)
	return st
}
