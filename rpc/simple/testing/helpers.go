// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"github.com/juju/juju/juju"
	"github.com/juju/juju/state"
)

func UUIDFromState(st *state.State) (string, error) {
	env, err := st.Environment()
	if err != nil {
		return "", err
	}
	return env.UUID(), nil
}

func HostnameFromAPIConn(conn *juju.APIConn) (string, error) {
	_, info, err := conn.Environ.StateInfo()
	if err != nil {
		return "", err
	}
	return info.Addrs[0], nil
}
