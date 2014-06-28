// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client

import (
	"github.com/juju/juju/state/api"
)

// Client represents the client-accessible part of the state.
type Client struct {
	st *api.State
}

func (c *Client) call(method string, params, result interface{}) error {
	return c.st.Call("Client", "", method, params, result)
}

// Close closes the Client's underlying State connection
// Client is unique among the api.State facades in closing its own State
// connection, but it is conventional to use a Client object without any access
// to its underlying state connection.
func (c *Client) Close() error {
	return c.st.Close()
}
