// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state_test

import (
	stdtesting "testing"
	"time"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/environmentserver/authentication"
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

func (s *BaseStateSuite) tryOpenState(info *authentication.MongoInfo) error {
	st, err := state.Open(info, s.DialOpts(), state.Policy(nil))
	if err == nil {
		st.Close()
	}
	return err
}

func (s *BaseStateSuite) setMongoPassword(c *gc.C, getEntity func(st *state.State) (entity, error)) {
	info := s.MongoInfo()
	st, err := state.Open(info, s.DialOpts(), state.Policy(nil))
	c.Assert(err, gc.IsNil)
	defer st.Close()
	// Turn on fully-authenticated mode.
	err = st.SetAdminMongoPassword("admin-secret")
	c.Assert(err, gc.IsNil)

	// Set the password for the entity
	ent, err := getEntity(st)
	c.Assert(err, gc.IsNil)
	err = ent.SetMongoPassword("foo")
	c.Assert(err, gc.IsNil)

	// Check that we cannot log in with the wrong password.
	info.Tag = ent.Tag()
	info.Password = "bar"
	err = s.tryOpenState(info)
	c.Assert(err, jc.Satisfies, errors.IsUnauthorized)

	// Check that we can log in with the correct password.
	info.Password = "foo"
	st1, err := state.Open(info, s.DialOpts(), state.Policy(nil))
	c.Assert(err, gc.IsNil)
	defer st1.Close()

	// Change the password with an entity derived from the newly
	// opened and authenticated state.
	ent, err = getEntity(st)
	c.Assert(err, gc.IsNil)
	err = ent.SetMongoPassword("bar")
	c.Assert(err, gc.IsNil)

	// Check that we cannot log in with the old password.
	info.Password = "foo"
	err = s.tryOpenState(info)
	c.Assert(err, jc.Satisfies, errors.IsUnauthorized)

	// Check that we can log in with the correct password.
	info.Password = "bar"
	err = s.tryOpenState(info)
	c.Assert(err, gc.IsNil)

	// Check that the administrator can still log in.
	info.Tag, info.Password = nil, "admin-secret"
	err = s.tryOpenState(info)
	c.Assert(err, gc.IsNil)

	// Remove the admin password so that the test harness can reset the state.
	err = st.SetAdminMongoPassword("")
	c.Assert(err, gc.IsNil)
}

type waiter interface {
	Wait() error
}

// testWatcherDiesWhenStateCloses calls the given function to start a watcher,
// closes the state and checks that the watcher dies with the expected error.
// The watcher should already have consumed the first
// event, otherwise the watcher's initialisation logic may
// interact with the closed state, causing it to return an
// unexpected error (often "Closed explictly").
func (s *BaseStateSuite) assertWatcherDiesWhenStateCloses(
	c *gc.C, startWatcher func(c *gc.C, st *state.State) waiter,
) {
	st, err := state.Open(s.MongoInfo(), s.DialOpts(), state.Policy(nil))
	c.Assert(err, gc.IsNil)
	watcher := startWatcher(c, st)
	err = st.Close()
	c.Assert(err, gc.IsNil)
	done := make(chan error)
	go func() {
		done <- watcher.Wait()
	}()
	select {
	case err := <-done:
		c.Assert(err, gc.Equals, state.ErrStateClosed)
	case <-time.After(testing.LongWait):
		c.Fatalf("watcher %T did not exit when state closed", watcher)
	}
}
