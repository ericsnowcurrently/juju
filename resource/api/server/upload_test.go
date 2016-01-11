// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package server_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	charmresource "gopkg.in/juju/charm.v6-unstable/resource"

	"github.com/juju/juju/resource"
	"github.com/juju/juju/resource/api/server"
)

type UploadSuite struct {
	BaseSuite

	timestamp time.Time

	req    *http.Request
	header http.Header
	resp   *stubHTTPResponseWriter
}

var _ = gc.Suite(&UploadSuite{})

func (s *UploadSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)

	s.timestamp = time.Now()

	method := "..."
	urlStr := "..."
	body := strings.NewReader("...")
	req, err := http.NewRequest(method, urlStr, body)
	c.Assert(err, jc.ErrorIsNil)

	s.req = req
	s.header = make(http.Header)
	s.resp = &stubHTTPResponseWriter{
		stub:         s.stub,
		returnHeader: s.header,
	}
}

func (s *UploadSuite) now() time.Time {
	s.stub.AddCall("CurrentTimestamp")
	s.stub.NextErr() // Pop one off.

	return s.timestamp
}

func (s *UploadSuite) TestHandleRequestOkay(c *gc.C) {
	content := "<some data>"
	res, _ := newResource(c, "spam", "a-user", content)
	res.Timestamp = s.timestamp.UTC()
	stored, _ := newResource(c, "spam", "", "")
	s.data.ReturnListResources = []resource.Resource{stored}
	uh := server.UploadHandler{
		Username:         "a-user",
		Store:            s.data,
		CurrentTimestamp: s.now,
	}
	req, body := newUploadRequest(c, "spam", "a-service", content)

	err := uh.HandleRequest(req)
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCallNames(c, "ListResources", "CurrentTimestamp", "SetResource")
	s.stub.CheckCall(c, 0, "ListResources", "a-service")
	s.stub.CheckCall(c, 2, "SetResource", "a-service", res, ioutil.NopCloser(body))
}

func (s *UploadSuite) TestHandleRequestSetResourceFailure(c *gc.C) {
	content := "<some data>"
	res, _ := newResource(c, "spam", "a-user", content)
	res.Timestamp = s.timestamp.UTC()
	stored, _ := newResource(c, "spam", "", "")
	s.data.ReturnListResources = []resource.Resource{stored}
	uh := server.UploadHandler{
		Username:         "a-user",
		Store:            s.data,
		CurrentTimestamp: s.now,
	}
	req, _ := newUploadRequest(c, "spam", "a-service", content)
	failure := errors.New("<failure>")
	s.stub.SetErrors(nil, nil, failure)

	err := uh.HandleRequest(req)

	c.Check(errors.Cause(err), gc.Equals, failure)
	s.stub.CheckCallNames(c, "ListResources", "CurrentTimestamp", "SetResource")
}

func (s *UploadSuite) TestReadResourceOkay(c *gc.C) {
	content := "<some data>"
	expected, _ := newResource(c, "spam", "a-user", content)
	expected.Timestamp = s.timestamp.UTC()
	stored, _ := newResource(c, "spam", "", "")
	s.data.ReturnListResources = []resource.Resource{stored}
	uh := server.UploadHandler{
		Username:         "a-user",
		Store:            s.data,
		CurrentTimestamp: s.now,
	}
	req, body := newUploadRequest(c, "spam", "a-service", content)

	uploaded, err := uh.ReadResource(req)
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCallNames(c, "ListResources", "CurrentTimestamp")
	s.stub.CheckCall(c, 0, "ListResources", "a-service")
	c.Check(uploaded, jc.DeepEquals, &server.UploadedResource{
		Service:  "a-service",
		Resource: expected,
		Data:     ioutil.NopCloser(body),
	})
}

func (s *UploadSuite) TestReadResourceBadContentType(c *gc.C) {
	uh := server.UploadHandler{
		Username:         "a-user",
		Store:            s.data,
		CurrentTimestamp: s.now,
	}
	req, _ := newUploadRequest(c, "spam", "a-service", "<some data>")
	req.Header.Set("Content-Type", "text/plain")

	_, err := uh.ReadResource(req)

	c.Check(err, gc.ErrorMatches, "unsupported content type .*")
	s.stub.CheckNoCalls(c)
}

func (s *UploadSuite) TestReadResourceNotFound(c *gc.C) {
	uh := server.UploadHandler{
		Username:         "a-user",
		Store:            s.data,
		CurrentTimestamp: s.now,
	}
	req, _ := newUploadRequest(c, "spam", "a-service", "<some data>")

	_, err := uh.ReadResource(req)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
	s.stub.CheckCallNames(c, "ListResources")
}

func (s *UploadSuite) TestReadResourceBadFingerprint(c *gc.C) {
	stored, _ := newResource(c, "spam", "", "")
	s.data.ReturnListResources = []resource.Resource{stored}
	uh := server.UploadHandler{
		Username:         "a-user",
		Store:            s.data,
		CurrentTimestamp: s.now,
	}
	req, _ := newUploadRequest(c, "spam", "a-service", "<some data>")
	query := req.URL.Query()
	query.Set("fingerprint", "bogus")
	req.URL.RawQuery = query.Encode()

	_, err := uh.ReadResource(req)

	c.Check(err, gc.ErrorMatches, "invalid fingerprint.*")
	s.stub.CheckCallNames(c, "ListResources")
}

func (s *UploadSuite) TestReadResourceBadSize(c *gc.C) {
	stored, _ := newResource(c, "spam", "", "")
	s.data.ReturnListResources = []resource.Resource{stored}
	uh := server.UploadHandler{
		Username:         "a-user",
		Store:            s.data,
		CurrentTimestamp: s.now,
	}
	req, _ := newUploadRequest(c, "spam", "a-service", "<some data>")
	req.Header.Set("Content-Length", "should-be-an-int")

	_, err := uh.ReadResource(req)

	c.Check(err, gc.ErrorMatches, "invalid size.*")
	s.stub.CheckCallNames(c, "ListResources")
}

func newUploadRequest(c *gc.C, name, service, content string) (*http.Request, io.Reader) {
	fp, err := charmresource.GenerateFingerprint([]byte(content))
	c.Assert(err, jc.ErrorIsNil)
	size := len(content)

	method := "PUT"
	urlStr := "https://api:17017/services/%s/resources/%s"
	urlStr += "?size=%d&fingerprint=%s"
	urlStr += "&:service=%s&:resource=%s" // ...added by the mux.
	urlStr = fmt.Sprintf(urlStr, service, name, size, fp.String(), service, name)
	body := strings.NewReader(content)
	req, err := http.NewRequest(method, urlStr, body)
	c.Assert(err, jc.ErrorIsNil)

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", fmt.Sprint(len(content)))

	return req, body
}