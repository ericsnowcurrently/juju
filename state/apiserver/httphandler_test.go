// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/instance"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/api"
	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/testing/factory"
)

type httpHandlerSuite struct {
	jujutesting.JujuConnSuite
	userTag    string
	password   string
	apiBinding string
	httpMethod string
	apiClient  *api.Client
	httpClient *http.Client
	noop       bool
}

func (s *httpHandlerSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	s.password = "password"
	user := s.Factory.MakeUser(factory.UserParams{Password: s.password})
	s.userTag = user.Tag().String()
	s.apiClient = s.APIState.Client()
	s.httpClient = utils.GetNonValidatingHTTPClient()
	s.noop = false
}

func (s *httpHandlerSuite) URL(c *gc.C, uuid string) *url.URL {
	if uuid == "" {
		uuid = s.apiClient.EnvironmentUUID()
	}
	URL := url.URL{
		Scheme: "https",
		Host:   s.APIState.Addr(),
		Path:   fmt.Sprintf("/environment/%s/%s", uuid, s.apiBinding),
	}
	return &URL
}

func (s *httpHandlerSuite) addMachine(c *gc.C, series, name string) (string, string) {
	machine, err := s.State.AddMachine(series, state.JobHostUnits)
	c.Assert(err, gc.IsNil)

	err = machine.SetProvisioned(instance.Id(name), "fake_nonce", nil)
	c.Assert(err, gc.IsNil)

	password, err := utils.RandomPassword()
	c.Assert(err, gc.IsNil)
	err = machine.SetPassword(password)
	c.Assert(err, gc.IsNil)

	tag := machine.Tag().String()
	return tag, password
}

func (s *httpHandlerSuite) checkAPIBinding(c *gc.C, invalidMethods ...string) bool {
	res := s.checkServedSecurely(c)
	for _, method := range invalidMethods {
		res = res && s.checkHTTPMethodInvalid(c, method)
	}

	res = res && s.checkRequiresAuth(c)
	res = res && s.checkAuthRequiresUser(c)

	res = res && s.checkEnvUUIDPathAllowed(c)
	res = res && s.checkRejectsWrongEnvUUIDPath(c)

	return res
}

//---------------------------
// request helpers

func (s *httpHandlerSuite) sendRequest(
	c *gc.C, tag, password, method, uri, contentType string, body io.Reader,
) (*http.Response, error) {
	if method == "" {
		method = s.httpMethod
	}
	if uri == "" {
		uri = s.URL(c, "").String()
		if s.noop {
			uri += "?noop=1"
		}
	} else if s.noop {
		if strings.Contains(uri, "?") {
			uri += "&noop=1"
		} else {
			uri += "?noop=1"
		}
	}
	req, err := http.NewRequest(method, uri, body)
	c.Assert(err, gc.IsNil)

	if tag != "" && password != "" {
		req.SetBasicAuth(tag, password)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return s.httpClient.Do(req)
}

func (s *httpHandlerSuite) authRequest(
	c *gc.C, method, uri, contentType string, body io.Reader,
) *http.Response {
	resp, err := s.sendRequest(c, s.userTag, s.password, method, uri, contentType, body)
	c.Assert(err, gc.IsNil)
	return resp
}

func (s *httpHandlerSuite) validRequest(c *gc.C) *http.Response {
	return s.authRequest(c, "", "", "", nil)
}

//---------------------------
// response checks

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

//---------------------------
// HTTP checks

func (s *httpHandlerSuite) checkServedSecurely(c *gc.C) bool {
	s.noop = true

	URL := s.URL(c, "")
	URL.Scheme = "http"
	_, err := s.sendRequest(c, "", "", "", URL.String(), "", nil)
	return c.Check(err, gc.ErrorMatches, `.*malformed HTTP response.*`)
}

func (s *httpHandlerSuite) checkHTTPMethodInvalid(c *gc.C, method string) bool {
	s.noop = true

	resp := s.authRequest(c, method, "", "", nil)
	return s.checkErrorResponse(c, resp, http.StatusMethodNotAllowed, fmt.Sprintf(`unsupported method: "%s"`, method))
}

//---------------------------
// path checks

func (s *httpHandlerSuite) checkLegacyPathDisallowed(c *gc.C) bool {
	s.noop = true

	URL := s.URL(c, "")
	URL.Path = "/" + s.apiBinding
	resp := s.authRequest(c, "", URL.String(), "", nil)
	return c.Check(resp.StatusCode, gc.Equals, http.StatusNotFound)
}

func (s *httpHandlerSuite) checkLegacyPathAllowed(c *gc.C) bool {
	s.noop = true

	URL := s.URL(c, "")
	URL.Path = "/" + s.apiBinding
	resp := s.authRequest(c, "", URL.String(), "", nil)
	return s.checkErrorResponse(c, resp, http.StatusInternalServerError, "no-op")
}

func (s *httpHandlerSuite) checkEnvUUIDPathAllowed(c *gc.C) bool {
	s.noop = true

	resp := s.validRequest(c)
	return s.checkErrorResponse(c, resp, http.StatusInternalServerError, "no-op")
}

func (s *httpHandlerSuite) checkRejectsWrongEnvUUIDPath(c *gc.C) bool {
	s.noop = true

	URL := s.URL(c, "dead-beef-123456")
	resp := s.authRequest(c, "", URL.String(), "", nil)
	return s.checkErrorResponse(c, resp, http.StatusNotFound, `unknown environment: "dead-beef-123456"`)
}

//---------------------------
// auth checks

func (s *httpHandlerSuite) checkRequiresAuth(c *gc.C) bool {
	s.noop = true

	resp, err := s.sendRequest(c, "", "", "", "", "", nil)
	c.Assert(err, gc.IsNil)
	return s.checkErrorResponse(c, resp, http.StatusUnauthorized, "unauthorized")
}

func (s *httpHandlerSuite) checkAuthRequiresUser(c *gc.C) bool {
	s.noop = true

	// Add a machine and try to login.
	tag, pw := s.addMachine(c, "quantal", "foo")
	resp, err := s.sendRequest(c, tag, pw, "", "", "", nil)
	c.Assert(err, gc.IsNil)
	res := s.checkErrorResponse(c, resp, http.StatusUnauthorized, "unauthorized")

	// Now try a user login.
	resp = s.validRequest(c)
	return res && s.checkErrorResponse(c, resp, http.StatusInternalServerError, "no-op")
}
