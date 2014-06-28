// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	ec "github.com/juju/juju/state/api/client.errorcodes"
)

type rawRPCData struct {
	File     io.Reader
	Mimetype string
}

func (c *Client) getRawRPCRequest(method string, args *url.Values, data *rawRPCData) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s", c.st.serverRoot)
	if args != nil {
		url += "?" + args.Encode()
	}

	var datafile io.Reader
	if data != nil {
		datafile = data.File
	}
	req, err := http.NewRequest("POST", url, datafile)
	if err != nil {
		return nil, ec.RequestNotCreated.Err(err, "")
	}

	req.SetBasicAuth(c.st.tag, c.st.password)

	if data != nil && data.Mimetype != "" {
		req.Header.Set("Content-Type", data.mimetype)
	}

	return req, nil
}

// BUG(dimitern) 2013-12-17 bug #1261780
// Due to issues with go 1.1.2, fixed later, we cannot use a
// regular TLS client with the CACert here, because we get "x509:
// cannot validate certificate for 127.0.0.1 because it doesn't
// contain any IP SANs". Once we use a later go version, this
// should be changed to connect to the API server with a regular
// HTTP+TLS enabled client, using the CACert (possily cached, like
// the tag and password) passed in api.Open()'s info argument.
func (c *Client) getRawRPCClient() *http.Client {
	return utils.GetNonValidatingHTTPClient()
}

func (c *Client) sendRawRPCRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, ec.RequestNotSent.Err(err, "")
	}

	return resp, nil
}

type rawRPCResult interface {
	ResultError() string // Returns an error message if there is one.
}

func (c *Client) handleRawRPCResponse(resp *http.Response, result rawRPCResult) error {
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		// API server is 1.16 or older, so request
		// is not supported; notify the client.
		return ec.MethodNotSupported.Err(nil, "")
	}

	// Now parse the response.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ec.ErrorDataNotRead.Err(err, "")
	}

	if err := json.Unmarshal(body, result); err != nil {
		return ec.ErrorDataNotUnpacked.Err(err, "")
	}

	err = result.ResultError()
	if err != nil {
		return err
	}

	return nil
}

// XXX Change args to be interface{}, a la the juju's RPC.
func (c *Client) callRaw(method string, args *url.Values, data *rawRPCData, result rawRPCResult) error {

	req, err := c.getRawRPCRequest(method, args, data)
	if req == nil {
		return err
	}

	client = c.getRawRPCClient()
	resp, err = c.sendRawRPCRequest(client, req)
	if err != nil {
		return err
	}

	err = c.handleRawRPCResponse(resp, result)
	if err != nil {
		return err
	}

	return nil
}
