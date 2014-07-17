// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"net/http"
	"os"

	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/instance"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/rpc/errors"
	"github.com/juju/juju/rpc/simple"
	"github.com/juju/juju/state"
	"github.com/juju/juju/testing"
	"github.com/juju/juju/testing/factory"
)

type HTTPReqSuite struct {
	jujutesting.JujuConnSuite
	testing.FilesSuite

	// XXX Change to using *simple.APIBinding.
	APIBinding string
	HTTPMethod string
	Mimetype   string
	APIArgs    string // We could also use *url.Values.
	HTTPClient simple.HTTPDoer

	userTag  string
	password string
}

func (s *HTTPReqSuite) SetUpSuite(c *gc.C) {
	s.JujuConnSuite.SetUpSuite(c)
	s.FilesSuite.SetUpSuite(c)
}

func (s *HTTPReqSuite) TearDownSuite(c *gc.C) {
	s.JujuConnSuite.TearDownSuite(c)
	s.FilesSuite.TearDownSuite(c)
}

func (s *HTTPReqSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	s.FilesSuite.SetUpTest(c)

	s.password = "password"
	user := s.Factory.MakeUser(factory.UserParams{Password: s.password})
	s.userTag = user.Tag().String()

	// Set defaults.
	s.HTTPMethod = "POST"
	s.HTTPClient = s.APIState.SecureHTTPClient("anything")
}

func (s *HTTPReqSuite) TearDownTest(c *gc.C) {
	s.JujuConnSuite.TearDownTest(c)
	s.FilesSuite.TearDownTest(c)
}

func (s *HTTPReqSuite) Hostname(c *gc.C) string {
	hostname, err := HostnameFromAPIConn(s.APIConn)
	c.Assert(err, gc.IsNil)
	return hostname
}

func (s *HTTPReqSuite) Binding(c *gc.C) *simple.APIBinding {
	binding := simple.APIBinding{
		Name:       s.APIBinding,
		HTTPMethod: s.HTTPMethod,
		Mimetype:   s.Mimetype,
	}
	return &binding
}

func (s *HTTPReqSuite) UUID(c *gc.C) string {
	uuid, err := UUIDFromState(s.State)
	c.Assert(err, gc.IsNil)
	return uuid
}

func (s *HTTPReqSuite) AuthHeader() http.Header {
	return utils.BasicAuthHeader(s.userTag, s.password)
}

func (s *HTTPReqSuite) AddMachine(
	c *gc.C, series string, id instance.Id,
) (
	*state.Machine, string,
) {
	nonce := "fake_nonce"

	machine, err := s.State.AddMachine(series, state.JobHostUnits)
	c.Check(err, gc.IsNil)
	err = machine.SetProvisioned(id, nonce, nil)
	c.Check(err, gc.IsNil)
	password, err := utils.RandomPassword()
	c.Check(err, gc.IsNil)
	err = machine.SetPassword(password)
	c.Check(err, gc.IsNil)
	return machine, password
}

//---------------------------
// requests

type TestRequest struct {
	*simple.APIRequest
	client simple.HTTPDoer
}

func (s *HTTPReqSuite) NewRequestPlain(c *gc.C) *TestRequest {
	hostname := s.Hostname(c)
	binding := s.Binding(c)
	uuid := s.UUID(c)

	req := simple.NewAPIRequest(hostname, binding, uuid)
	return &TestRequest{req, s.HTTPClient}
}

func (s *HTTPReqSuite) NewRequest(c *gc.C) *TestRequest {
	req := s.NewRequestPlain(c)
	req.SetRawQuery(s.APIArgs)
	req.SetAuth(s.userTag, s.password)
	return req
}

func (tr *TestRequest) SetFile(c *gc.C, path string) *os.File {
	// If necessary, we could guess the mimetype from the path.
	var file *os.File
	// A missing path means just set the header.
	if path != "" {
		var err error
		file, err = os.Open(path)
		c.Assert(err, gc.IsNil)
	}
	tr.SetPayload(file)
	return file
}

func (tr *TestRequest) Send(c *gc.C) (*http.Response, error) {
	return tr.APIRequest.Send(tr.client)
}

func (tr *TestRequest) SendValid(c *gc.C) *http.Response {
	resp, err := tr.Send(c)
	c.Assert(err, gc.IsNil)
	return resp
}

func (s *HTTPReqSuite) Send(c *gc.C) (*http.Response, error) {
	req := s.NewRequest(c)
	return req.Send(c)
}

func (s *HTTPReqSuite) SendValid(c *gc.C) *http.Response {
	req := s.NewRequest(c)
	return req.SendValid(c)
}

//---------------------------
// response assertions

func (s *HTTPReqSuite) CheckStatusCode(c *gc.C, resp *http.Response, statusCode int) {
	if resp != nil {
		c.Check(resp.StatusCode, gc.Equals, statusCode)
	}
}

func (s *HTTPReqSuite) CheckResponse(
	c *gc.C, resp *http.Response, statusCode int, mimetype string,
) {
	c.Check(resp.StatusCode, gc.Equals, statusCode)
	ctype := resp.Header.Get("Content-Type")
	c.Check(ctype, gc.Equals, mimetype)
}

func (s *HTTPReqSuite) CheckJSONResponse(
	c *gc.C, resp *http.Response, result interface{},
) {
	s.CheckResponse(c, resp, http.StatusOK, "application/json")
	failure, err := simple.ReadJSONResponse(resp, result)
	c.Assert(err, gc.IsNil)
	c.Check(failure, gc.IsNil)
}

func (s *HTTPReqSuite) CheckErrorResponse(
	c *gc.C, resp *http.Response, statusCode int, msg string,
	result errors.FailableResult,
) {
	s.CheckResponse(c, resp, statusCode, "application/json")
	failure, err := simple.ReadJSONResponse(resp, result)
	c.Assert(err, gc.IsNil)
	c.Check(failure, gc.ErrorMatches, msg)
}

func (s *HTTPReqSuite) CheckFileResponse(
	c *gc.C, resp *http.Response, expected, mimetype string,
) {
	s.CheckResponse(c, resp, http.StatusOK, mimetype)
	body, err := simple.ReadResponse(resp)
	c.Assert(err, gc.IsNil)
	c.Check(string(body), gc.Equals, expected)
}

//---------------------------
// HTTP assertions

func (s *HTTPReqSuite) CheckServedSecurely(c *gc.C) {
	req := s.NewRequest(c)
	req.Host.Scheme = "http"

	_, err := req.Send(c)
	c.Check(err, gc.ErrorMatches, `.*malformed HTTP response.*`)
}

func (s *HTTPReqSuite) CheckHTTPMethodInvalid(
	c *gc.C, method string, result errors.FailableResult,
) {
	req := s.NewRequest(c)
	req.Binding.HTTPMethod = method
	resp, err := req.Send(c)
	s.CheckStatusCode(c, resp, http.StatusMethodNotAllowed)
	c.Check(err, gc.ErrorMatches, "API method not supported by server")
}

func (s *HTTPReqSuite) checkLegacyPath(c *gc.C, expected string) {
	req := s.NewRequest(c)
	req.EnvUUID = ""
	resp, err := req.Send(c)

	if expected != "" {
		c.Assert(err, gc.IsNil)
		s.CheckFileResponse(c, resp, expected, "text/plain; charset=utf-8")
	} else {
		c.Check(resp.StatusCode, gc.Equals, http.StatusNotFound)
	}
}

func (s *HTTPReqSuite) CheckLegacyPathAvailable(c *gc.C, expected string) {
	s.checkLegacyPath(c, expected)
}

func (s *HTTPReqSuite) CheckLegacyPathUnavailable(c *gc.C) {
	s.checkLegacyPath(c, "")
}

//---------------------------
// auth assertions

func (s *HTTPReqSuite) CheckRequiresAuth(
	c *gc.C, result errors.FailableResult,
) {
	req := s.NewRequest(c)
	req.SetAuth("", "")
	resp, err := req.Send(c)
	s.CheckStatusCode(c, resp, http.StatusUnauthorized)
	c.Check(err, gc.ErrorMatches, "unauthorized")
}

func (s *HTTPReqSuite) CheckRequiresUser(
	c *gc.C, result errors.FailableResult,
) {
	// Try to log in unsuccessfully.
	machine, password := s.AddMachine(c, "quantal", "foo")
	req := s.NewRequest(c)
	req.SetAuth(machine.Tag().String(), password)
	resp, err := req.Send(c)
	s.CheckStatusCode(c, resp, http.StatusUnauthorized)
	c.Check(err, gc.ErrorMatches, "unauthorized")

	// Try to log in successfully.
	// (With an invalid method so we don't actually do anything.)
	s.CheckHTTPMethodInvalid(c, "PUT", result)
}

//---------------------------
// environment assertions

func (s *HTTPReqSuite) CheckBindingAllowsEnvUUIDPath(c *gc.C, expected string) {
	resp := s.SendValid(c)

	s.CheckFileResponse(c, resp, expected, "text/plain; charset=utf-8")
}

func (s *HTTPReqSuite) checkBindingWrongEnv(
	c *gc.C, result errors.FailableResult,
) {
	req := s.NewRequest(c)
	req.EnvUUID = "dead-beef-123456"
	resp, err := req.Send(c)
	s.CheckStatusCode(c, resp, http.StatusNotFound)
	c.Check(err, gc.ErrorMatches, `unknown environment: "dead-beef-123456"`)
}
