// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/juju/names"
	"github.com/juju/utils"
)

// XXX Add support for streaming data and upload/download rather than
// just passing JSON.

// Client represents the client-accessible part of the state.
type Client struct {
}

// NetworksSpecification holds the enabled and disabled networks for a
// service.
type NetworksSpecification struct {
	Enabled  []string
	Disabled []string
}

func (c *Client) call(method string, params, result interface{}) error {
	return c.st.Call("Client", "", method, params, result)
}

// Status returns the status of the API server.
func (c *Client) Status(patterns []string) (*Status, error) {
	var result Status
	p := params.StatusParams{Patterns: patterns}
	if err := c.call("FullStatus", p, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Close closes the Client's underlying API resources.
func (c *Client) Close() error {
	return nil
}
