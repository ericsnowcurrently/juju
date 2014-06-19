// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/juju/utils"

	"github.com/juju/juju/state/api/params"
)

// Authentication

type Auth interface {
	Set(req *http.Request) error
}

type BasicAuth struct {
	username string
	password string
}

func (a *BasicAuth) Set(req *http.Request) error {
	req.SetBasicAuth(a.username, a.password)
	return nil
}

// Raw Requests

type rawRPCRequest struct {
	Method   string
	Args     url.Values
	Auth     Auth
	Mimetype string
	Payload  *os.File
}

func (r *rawRPCRequest) GetArg(name string) (string, error) {
	if r.Args == nil {
		return "", fmt.Errorf("not found")
	}
	values, found := r.Args[name]
	if !found {
		return "", fmt.Errorf("not found")
	}
	return values[0], nil
}

func (r *rawRPCRequest) SetArg(name, value string) {
	if r.Args == nil {
		r.Args = url.Values{}
	}
	r.Args.Set(name, value)
}

func (r *rawRPCRequest) SetPayload(payload *os.File, mimetype string) {
	r.Payload = payload
	r.Mimetype = mimetype
}

func (r *rawRPCRequest) SetBasicAuth(username, password string) {
	r.Auth = &BasicAuth{username, password}
}

func (r *rawRPCRequest) URL(base string) string {
	rawurl := fmt.Sprintf("%s/%s", base, r.Method)
	if r.Args != nil {
		rawurl += "?" + r.Args.Encode()
	}
	return rawurl
}

func (r *rawRPCRequest) Request(urlbase string) (*http.Request, error) {
	req, err := http.NewRequest("POST", r.URL(urlbase), r.Payload)
	if err != nil {
		return nil, err
	}

	if r.Auth != nil {
		r.Auth.Set(req)
	}

	if r.Mimetype != "" {
		req.Header.Set("Content-Type", r.Mimetype)
	}

	return req, nil
}

func (r *rawRPCRequest) Send(urlbase string) (*rawRPCResponse, error) {
	req, err := r.Request(urlbase)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %v", err)
	}

	// BUG(dimitern) 2013-12-17 bug #1261780
	// Due to issues with go 1.1.2, fixed later, we cannot use a
	// regular TLS client with the CACert here, because we get "x509:
	// cannot validate certificate for 127.0.0.1 because it doesn't
	// contain any IP SANs". Once we use a later go version, this
	// should be changed to connect to the API server with a regular
	// HTTP+TLS enabled client, using the CACert (possily cached, like
	// the tag and password) passed in api.Open()'s info argument.
	resp, err := utils.GetNonValidatingHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		// API server is likely too old so the method
		// is not supported; notify the client.
		return nil, &params.Error{
			Message: "method not supported by the API server",
			Code:    params.CodeNotImplemented,
		}
	}

	return (*rawRPCResponse)(resp), nil
}

// Raw Responses

type rawRPCResponse http.Response

func (r *rawRPCResponse) UnpackJSON(target interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("cannot read response: %v", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("cannot unmarshal response: %v", err)
	}
	return nil
}
