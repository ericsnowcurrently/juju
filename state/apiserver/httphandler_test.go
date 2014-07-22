// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/testing/factory"
)

type httpHandlerSuite struct {
	jujutesting.JujuConnSuite
	userTag            string
	password           string
	archiveContentType string
}

func (s *httpHandlerSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	s.password = "password"
	user := s.Factory.MakeUser(factory.UserParams{Password: s.password})
	s.userTag = user.Tag().String()
}

func (s *httpHandlerSuite) baseURL(c *gc.C) *url.URL {
	info := s.APIInfo(c)
	return &url.URL{
		Scheme: "https",
		Host:   info.Addrs[0],
		Path:   "",
	}
}

func (s *httpHandlerSuite) sendRequest(c *gc.C, tag, password, method, uri, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, uri, body)
	c.Assert(err, gc.IsNil)
	if tag != "" && password != "" {
		req.SetBasicAuth(tag, password)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return utils.GetNonValidatingHTTPClient().Do(req)
}

func (s *httpHandlerSuite) authRequest(c *gc.C, method, uri, contentType string, body io.Reader) (*http.Response, error) {
	return s.sendRequest(c, s.userTag, s.password, method, uri, contentType, body)
}

func (s *httpHandlerSuite) checkResponse(
	c *gc.C, resp *http.Response, statusCode int, mimetype string,
) bool {
	res := c.Check(resp.StatusCode, gc.Equals, statusCode)
	ctype := resp.Header.Get("Content-Type")
	res = res && c.Check(ctype, gc.Equals, mimetype)
	return res
}

func (s *httpHandlerSuite) checkErrorResponse(
	c *gc.C, resp *http.Response, statusCode int, msg string,
) bool {
	res := s.checkResponse(c, resp, statusCode, "application/json")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, gc.IsNil)

	var result params.Error
	err = json.Unmarshal(body, &result)
	c.Assert(err, gc.IsNil)
	res = res && c.Check(&result, gc.ErrorMatches, msg)
	return res
}
