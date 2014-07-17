// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package simple_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/rpc/simple"
	coretesting "github.com/juju/juju/testing"
)

type httpreqSuite struct {
	coretesting.BaseSuite
}

var _ = gc.Suite(&httpreqSuite{})

//---------------------------
// test helpers

type FakeData struct {
	Raw io.Reader
	Err error
}

func NewFakeData(data string) *FakeData {
	raw := bytes.NewBufferString(data)
	return &FakeData{Raw: raw}
}

func InvalidData(msg string) *FakeData {
	return &FakeData{Err: fmt.Errorf(msg)}
}

func (d *FakeData) Read(p []byte) (n int, err error) {
	if d.Err != nil {
		return 0, d.Err
	}
	return d.Raw.Read(p)
}

func (d *FakeData) Close() error {
	return nil
}

type FakeResult struct{ Error string }

func (r *FakeResult) Err() error {
	if r.Error != "" {
		return fmt.Errorf(r.Error)
	}

	return nil
}

type FakeHTTPClient struct {
	Code int
	Data io.Reader
	Err  error
}

func (c FakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.Err != nil {
		return nil, c.Err
	}

	code := c.Code
	if code <= 0 {
		code = http.StatusOK
	}

	data := c.Data
	if data == nil {
		data = bytes.NewBufferString("")
	}

	resp := http.Response{
		StatusCode: code,
		Body:       &FakeData{Raw: data},
	}

	return &resp, nil
}

//---------------------------
// Do() tests

func (s *httpreqSuite) TestSendRequestValidNoData(c *gc.C) {
	client := FakeHTTPClient{}
	resp, err := simple.SendRequest(&client, nil)
	data, _ := ioutil.ReadAll(resp.Body)

	c.Assert(err, gc.IsNil)
	c.Assert(string(data), gc.Equals, "")
}

func (s *httpreqSuite) TestSendRequestValidData(c *gc.C) {
	client := FakeHTTPClient{Data: bytes.NewBufferString("raw data")}
	resp, err := simple.SendRequest(&client, nil)
	data, _ := ioutil.ReadAll(resp.Body)

	c.Assert(err, gc.IsNil)
	c.Assert(string(data), gc.Equals, "raw data")
}

func (s *httpreqSuite) TestSendRequestRequestSendFailed(c *gc.C) {
	client := FakeHTTPClient{Err: fmt.Errorf("failed!")}
	_, err := simple.SendRequest(&client, nil)

	c.Assert(err, gc.ErrorMatches, "could not send API request: .*")
}

func (s *httpreqSuite) TestSendRequestMethodNotSupported(c *gc.C) {
	client := FakeHTTPClient{Code: http.StatusMethodNotAllowed}
	_, err := simple.SendRequest(&client, nil)

	c.Assert(err, gc.ErrorMatches, "API method not supported by server")
}

// tests for method failures returned by the API server

func (s *httpreqSuite) TestSendRequestUnreadableErrorData(c *gc.C) {
	client := FakeHTTPClient{
		Data: InvalidData("invalid!"),
		Code: http.StatusInternalServerError,
	}
	_, err := simple.SendRequest(&client, nil)

	c.Assert(err, gc.ErrorMatches, "could not unpack result: .*")
}

func (s *httpreqSuite) TestSendRequestBadErrorData(c *gc.C) {
	client := FakeHTTPClient{
		Data: bytes.NewBufferString("not valid JSON"),
		Code: http.StatusInternalServerError,
	}
	_, err := simple.SendRequest(&client, nil)

	c.Assert(err, gc.ErrorMatches, "could not unpack result: .*")
}

func (s *httpreqSuite) TestSendRequestFailedRemotely(c *gc.C) {
	client := FakeHTTPClient{
		Data: bytes.NewBufferString(`{"Message": "failed!"}`),
		Code: http.StatusInternalServerError,
	}
	_, err := simple.SendRequest(&client, nil)

	c.Assert(err, gc.ErrorMatches, "failed!")
}

func (s *httpreqSuite) TestSendRequestBadStatusCode(c *gc.C) {
	client := FakeHTTPClient{
		Data: bytes.NewBufferString(`{}`),
		Code: http.StatusInternalServerError,
	}
	_, err := simple.SendRequest(&client, nil)
	c.Assert(err, gc.NotNil)

	c.Assert(err.Error(), gc.Equals, "")
}
