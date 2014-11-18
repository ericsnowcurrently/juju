// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"github.com/juju/names"
	gitjujutesting "github.com/juju/testing"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/mongo"
	"github.com/juju/juju/state"
	"github.com/juju/juju/testing"
)

// NewMongoInfo returns information suitable for
// connecting to the testing state server's mongo database.
func NewMongoInfo() *mongo.MongoInfo {
	return &mongo.MongoInfo{
		Info: mongo.Info{
			Addrs:  []string{gitjujutesting.MgoServer.Addr()},
			CACert: testing.CACert,
		},
	}
}

// NewDialOpts returns configuration parameters for
// connecting to the testing state server.
func NewDialOpts() mongo.DialOpts {
	return mongo.DialOpts{
		Timeout: testing.LongWait,
	}
}

// Initialize initializes the state and returns it. If state was not
// already initialized, and cfg is nil, the minimal default environment
// configuration will be used.
func Initialize(c *gc.C, owner names.UserTag, cfg *config.Config, policy state.Policy) *state.State {
	if cfg == nil {
		cfg = testing.EnvironConfig(c)
	}
	mgoInfo := NewMongoInfo()
	dialOpts := NewDialOpts()

	st, err := state.Initialize(owner, mgoInfo, cfg, dialOpts, policy)
	c.Assert(err, gc.IsNil)
	return st
}

func newState(c *gc.C, policy state.Policy) (*state.State, names.UserTag) {
	owner := names.NewLocalUserTag("test-admin")
	cfg := testing.EnvironConfig(c)
	return Initialize(c, owner, cfg, policy), owner
}

// NewState initializes a new state with default values for testing and
// returns it.
func NewState(c *gc.C) (*state.State, names.UserTag) {
	policy := MockPolicy{}
	return newState(c, &policy)
}

// NewStateWithoutPolicy initializes a new state with default values for
// testing and returns it.  The policy is set to nil.
func NewStateWithoutPolicy(c *gc.C) (*state.State, names.UserTag) {
	return newState(c, nil)
}
