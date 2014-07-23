// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/environs/storage"
	"github.com/juju/juju/environs/tools"
	toolstesting "github.com/juju/juju/environs/tools/testing"
	"github.com/juju/juju/state/api/params"
	coretools "github.com/juju/juju/tools"
	"github.com/juju/juju/version"
)

type toolsSuite struct {
	httpHandlerSuite
}

var _ = gc.Suite(&toolsSuite{})

func (s *toolsSuite) SetUpSuite(c *gc.C) {
	s.httpHandlerSuite.SetUpSuite(c)
	s.apiBinding = "tools"
	s.httpMethod = "POST"
	s.dataMimetype = "application/x-tar-gz"
}

func (s *toolsSuite) toolsURL(c *gc.C, version, series string) string {
	URL := s.URL(c, "")

	query := ""
	if version != "" {
		if query != "" {
			query += "&"
		}
		query += "binaryVersion=" + version
	}
	if series != "" {
		if query != "" {
			query += "&"
		}
		query += "series=" + series
	}
	URL.RawQuery = query

	return URL.String()
}

type fakeTools struct {
	ToolsSet coretools.List
	Version  version.Binary
	Filename string
	Storage  storage.Storage
}

func (s *toolsSuite) newFakeTools(c *gc.C) *fakeTools {
	localStorage := c.MkDir()

	vers := version.MustParseBinary("1.9.0-quantal-amd64")

	versionStrings := []string{vers.String()}
	expectedTools := toolstesting.MakeToolsWithCheckSum(
		c, localStorage, "releases", versionStrings)

	toolsFile := tools.StorageName(vers)

	fake := fakeTools{
		ToolsSet: expectedTools,
		Version:  vers,
		Filename: filepath.Join(localStorage, toolsFile),
		Storage:  s.Environ.Storage(),
	}
	return &fake
}

func (t *fakeTools) Tools() *coretools.Tools {
	return t.ToolsSet[0]
}

func (t *fakeTools) SetURL() error {
	toolsURL, err := t.Storage.URL(tools.StorageName(t.Version))
	if err != nil {
		return err
	}
	agentTools := t.Tools()
	agentTools.URL = toolsURL
	return nil
}

func (t *fakeTools) Get() (io.Reader, error) {
	return t.Storage.Get(tools.StorageName(t.Version))
}

func (t *fakeTools) ReadStored() ([]byte, error) {
	reader, err := t.Get()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}

func (t *fakeTools) ReadActual() ([]byte, error) {
	return ioutil.ReadFile(t.Filename)
}

func (s *toolsSuite) checkToolsResponse(
	c *gc.C, resp *http.Response, fake *fakeTools,
) bool {
	err := fake.SetURL()
	c.Assert(err, gc.IsNil)

	var toolsResult params.ToolsResult
	res := s.checkResult(c, resp, &toolsResult)
	res = c.Check(toolsResult.Error, gc.IsNil) && res

	agentTools := fake.Tools()
	return c.Check(toolsResult.Tools, gc.DeepEquals, agentTools) && res
}

func (s *toolsSuite) checkTools(c *gc.C, fake *fakeTools) bool {
	uploadedData, err := fake.ReadStored()
	c.Assert(err, gc.IsNil)
	expectedData, err := fake.ReadActual()
	c.Assert(err, gc.IsNil)

	return c.Check(uploadedData, gc.DeepEquals, expectedData)
}

//---------------------------
// tests

func (s *toolsSuite) TestToolsAPIBinding(c *gc.C) {
	s.checkAPIBinding(c, "PUT", "GET")
	s.checkLegacyPathAllowed(c)
}

func (s *toolsSuite) TestToolsUploadValid(c *gc.C) {
	// Make some fake tools.
	fake := s.newFakeTools(c)
	// Now try uploading them.
	URL := s.toolsURL(c, fake.Version.String(), "")
	resp := s.uploadRequest(c, URL, true, fake.Filename)

	// Check the response.
	s.checkToolsResponse(c, resp, fake)

	// Check the contents.
	s.checkTools(c, fake)
}

func (s *toolsSuite) TestToolsUploadRequiresVersion(c *gc.C) {
	resp := s.validRequest(c)
	s.checkErrorResponse(c, resp, http.StatusBadRequest, "expected binaryVersion argument")
}

func (s *toolsSuite) TestToolsUploadFailsWithNoTools(c *gc.C) {
	// Create an empty file.
	tempFile, err := ioutil.TempFile(c.MkDir(), "tools")
	c.Assert(err, gc.IsNil)

	URL := s.toolsURL(c, "1.18.0-quantal-amd64", "")
	resp := s.uploadRequest(c, URL, true, tempFile.Name())
	s.checkErrorResponse(c, resp, http.StatusBadRequest, "no tools uploaded")
}

func (s *toolsSuite) TestToolsUploadFailsWithInvalidContentType(c *gc.C) {
	// Create an empty file.
	tempFile, err := ioutil.TempFile(c.MkDir(), "tools")
	c.Assert(err, gc.IsNil)

	// Now try with the default Content-Type.
	URL := s.toolsURL(c, "1.18.0-quantal-amd64", "")
	resp := s.uploadRequest(c, URL, false, tempFile.Name())
	s.checkErrorResponse(c, resp, http.StatusBadRequest, "expected Content-Type: application/x-tar-gz, got: application/octet-stream")
}

func (s *toolsSuite) TestToolsUploadFakeSeries(c *gc.C) {
	// Make some fake tools.
	fake := s.newFakeTools(c)
	// Now try uploading them.
	URL := s.toolsURL(c, fake.Version.String(), "precise,trusty")
	resp := s.uploadRequest(c, URL, true, fake.Filename)

	// Check the response.
	s.checkToolsResponse(c, resp, fake)

	// Check the contents.
	for _, series := range []string{"precise", "quantal", "trusty"} {
		fake.Version.Series = series
		s.checkTools(c, fake)
	}
}
