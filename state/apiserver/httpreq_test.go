// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/rpc/errors"
	"github.com/juju/juju/rpc/simple"
	simpleTesting "github.com/juju/juju/rpc/simple/testing"
)

// TODO (ericsnow) BUG#1336542 Update the following to use the common code:
// charms_test.go
// tools_test.go
// debuglog_test.go

type authHttpSuite struct {
	simpleTesting.HTTPReqSuite
}

//---------------------------
// legacy test helpers

func (s *authHttpSuite) baseURL() (*url.URL, error) {
	_, info, err := s.APIConn.Environ.StateInfo()
	if err != nil {
		return nil, err
	}
	baseurl := &url.URL{
		Scheme: "https",
		Host:   info.Addrs[0],
		Path:   "",
	}
	return baseurl, nil
}

func (s *authHttpSuite) legacySend(c *gc.C, tr *simpleTesting.TestRequest) (*http.Response, error) {
	req, err := tr.Raw()
	c.Assert(err, gc.IsNil)
	return s.HTTPClient.Do(req)
}

func (s *authHttpSuite) legacyRequest(c *gc.C, method, uri string) *simpleTesting.TestRequest {
	URL, err := url.Parse(uri)
	c.Assert(err, gc.IsNil)

	req := s.NewRequest(c)
	err = req.UpdateFromURL(URL)
	c.Assert(err, gc.IsNil)

	req.Binding.HTTPMethod = method

	return req
}

func (s *authHttpSuite) sendRequest(c *gc.C, tag, password, method, uri, contentType string, body io.Reader) (*http.Response, error) {
	req := s.legacyRequest(c, method, uri)
	req.SetAuth(tag, password)
	req.Payload = &simple.HTTPPayload{body, contentType}
	return s.legacySend(c, req)
}

func (s *authHttpSuite) authRequest(c *gc.C, method, uri, contentType string, body io.Reader) (*http.Response, error) {
	req := s.legacyRequest(c, method, uri)
	req.Payload = &simple.HTTPPayload{body, contentType}
	return s.legacySend(c, req)
}

func (s *authHttpSuite) uploadRequest(c *gc.C, uri string, asZip bool, path string) (*http.Response, error) {
	req := s.legacyRequest(c, s.HTTPMethod, uri)
	file := req.SetFile(c, path)
	defer file.Close()
	if !asZip {
		req.Payload.Mimetype = ""
	}
	return s.legacySend(c, req)
}

func (s *authHttpSuite) assertErrorResponse(c *gc.C, resp *http.Response, expCode int, expError string, result errors.FailableResult) {
	body := assertResponse(c, resp, expCode, "application/json")
	jsonResponse(c, body, result)
	c.Check(result.Err(), gc.ErrorMatches, expError)
}

func assertResponse(c *gc.C, resp *http.Response, expCode int, expContentType string) []byte {
	c.Check(resp.StatusCode, gc.Equals, expCode)
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, gc.IsNil)
	ctype := resp.Header.Get("Content-Type")
	c.Assert(ctype, gc.Equals, expContentType)
	return body
}

func jsonResponse(c *gc.C, body []byte, result errors.FailableResult) {
	err := json.Unmarshal(body, &result)
	c.Assert(err, gc.IsNil)
}
