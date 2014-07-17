// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"net/http"
	"net/url"

	"github.com/juju/juju/rpc/simple"
)

func (c *Client) rawHTTPClient() simple.HTTPDoer {
	return c.st.SecureHTTPClient("anything")
}

func (c *Client) newAPIRequest(binding *simple.APIBinding) (*simple.APIRequest, error) {
	URL, err := url.Parse(c.st.serverRoot)
	if err != nil {
		return nil, err
	}
	envinfo, err := c.EnvironmentInfo()
	if err != nil {
		return nil, err
	}
	req := simple.NewAPIRequest(URL.Host, binding, envinfo.UUID)

	req.SetAuth(c.st.tag, c.st.password)

	return req, nil
}

/*
TODO(ericsnow) 2014-07-01 bug #1336542
Client.AddLocalCharm() and Client.UploadTools() should
be updated to use this method (and this should be adapted to
accommodate them.  That will include adding parameters for "args" and
"payload".
*/
func (c *Client) sendHTTPRequest(binding *simple.APIBinding) (*http.Response, error) {
	req, err := c.newAPIRequest(binding)
	if err != nil {
		return nil, err
	}
	client := c.rawHTTPClient()
	return req.Send(client)
}
