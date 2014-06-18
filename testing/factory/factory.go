// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package factory

import (
	"fmt"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state"
)

type Factory struct {
	st    *state.State
	c     *gc.C
	index int
}

func NewFactory(st *state.State, c *gc.C) *Factory {
	return &Factory{st: st, c: c}
}

type UserParams struct {
	Username    string
	DisplayName string
	Password    string
	Creator     string
}

func (factory *Factory) UniqueInteger() int {
	factory.index++
	return factory.index
}

func (factory *Factory) UniqueString(prefix string) string {
	if prefix == "" {
		prefix = "no-prefix"
	}
	return fmt.Sprintf("%s-%d", prefix, factory.UniqueInteger())
}

// MakeAnyUser will create a user with no specified values.
func (factory *Factory) MakeAnyUser() *state.User {
	return factory.MakeUser(UserParams{})
}

// MakeUser will create a user with values defined by the params.
// For attributes of UserParams that are the default empty values,
// some meaningful valid values are used instead.
func (factory *Factory) MakeUser(params UserParams) *state.User {
	if params.Username == "" {
		params.Username = factory.UniqueString("username")
	}
	if params.DisplayName == "" {
		params.DisplayName = factory.UniqueString("display name")
	}
	if params.Password == "" {
		params.Password = "password"
	}
	if params.Creator == "" {
		params.Creator = "admin"
	}
	user, err := factory.st.AddUser(
		params.Username, params.DisplayName, params.Password, params.Creator)
	factory.c.Assert(err, gc.IsNil)
	return user
}
