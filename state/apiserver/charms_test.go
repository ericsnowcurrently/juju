// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/juju/charm"
	charmtesting "github.com/juju/charm/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/environs"
	"github.com/juju/juju/state/api/params"
)

//---------------------------
// charms base suite

type charmsSuite struct {
	httpHandlerSuite
}

func (s *charmsSuite) SetUpSuite(c *gc.C) {
	s.httpHandlerSuite.SetUpSuite(c)
	s.apiBinding = "charms"
}

func (s *charmsSuite) uploadBundle(
	c *gc.C, filename, series string,
) *http.Response {
	URL := s.URL(c, "")
	URL.RawQuery = "series=" + series
	return s.uploadRequest(c, URL.String(), true, filename)
}

//---------------------------
// charms common

type charmsCommonSuite struct {
	charmsSuite
}

var _ = gc.Suite(&charmsCommonSuite{})

func (s *charmsCommonSuite) SetUpSuite(c *gc.C) {
	s.charmsSuite.SetUpSuite(c)
	s.httpMethod = "GET"
}

func (s *charmsCommonSuite) TestCharmsAPIBinding(c *gc.C) {
	s.checkAPIBinding(c, "PUT")
	s.checkLegacyPathAllowed(c)
}

//---------------------------
// charms upload

type charmsUploadSuite struct {
	charmsSuite
}

var _ = gc.Suite(&charmsUploadSuite{})

func (s *charmsUploadSuite) SetUpSuite(c *gc.C) {
	s.charmsSuite.SetUpSuite(c)
	s.httpMethod = "POST"
	s.dataMimetype = "application/zip"
}

func (s *charmsUploadSuite) checkUploadResponse(
	c *gc.C, resp *http.Response, expCharmURL *charm.URL,
) bool {
	var result params.CharmsResponse
	res := s.checkResult(c, resp, &result)

	res = c.Check(result.Error, gc.Equals, "") && res
	return c.Check(result.CharmURL, gc.Equals, expCharmURL.String()) && res
}

func (s *charmsUploadSuite) TestCharmsUploadRequiresSeries(c *gc.C) {
	resp := s.urlRequest(c, nil)
	s.checkErrorResponse(c, resp, http.StatusBadRequest,
		"expected series=URL argument")
}

func (s *charmsUploadSuite) TestCharmsUploadFailsWithInvalidZip(c *gc.C) {
	// Create an empty file.
	tempFile, err := ioutil.TempFile(c.MkDir(), "charm")
	c.Assert(err, gc.IsNil)
	filename := tempFile.Name()

	// Pretend we upload a zip by setting the Content-Type, so we can
	// check the error at extraction time later.
	resp := s.uploadBundle(c, filename, "quantal")
	s.checkErrorResponse(c, resp, http.StatusBadRequest,
		"cannot open charm archive: zip: not a valid zip file")

	// Now try with the default Content-Type.
	URL := s.URL(c, "")
	URL.RawQuery = "series=quantal"
	resp = s.uploadRequest(c, URL.String(), false, filename)
	s.checkErrorResponse(c, resp, http.StatusBadRequest,
		"expected Content-Type: application/zip, got: application/octet-stream")
}

func (s *charmsUploadSuite) TestCharmsUploadBumpsRevision(c *gc.C) {
	// Add the dummy charm with revision 1.
	ch := charmtesting.Charms.Bundle(c.MkDir(), "dummy")
	curl := charm.MustParseURL(
		fmt.Sprintf("local:quantal/%s-%d", ch.Meta().Name, ch.Revision()),
	)
	bundleURL, err := url.Parse("http://bundles.testing.invalid/dummy-1")
	c.Assert(err, gc.IsNil)
	_, err = s.State.AddCharm(ch, curl, bundleURL, "dummy-1-sha256")
	c.Assert(err, gc.IsNil)

	// Now try uploading the same revision and verify it gets bumped,
	// and the BundleURL and BundleSha256 are calculated.
	resp := s.uploadBundle(c, ch.Path, "quantal")
	expectedURL := charm.MustParseURL("local:quantal/dummy-2")
	s.checkUploadResponse(c, resp, expectedURL)
	sch, err := s.State.Charm(expectedURL)
	c.Assert(err, gc.IsNil)
	c.Assert(sch.URL(), gc.DeepEquals, expectedURL)
	c.Assert(sch.Revision(), gc.Equals, 2)
	c.Assert(sch.IsUploaded(), jc.IsTrue)
	// No more checks for these two here, because they
	// are verified in TestCharmsUploadRespectsLocalRevision.
	c.Assert(sch.BundleURL(), gc.Not(gc.Equals), "")
	c.Assert(sch.BundleSha256(), gc.Not(gc.Equals), "")
}

func (s *charmsUploadSuite) TestCharmsUploadRespectsLocalRevision(c *gc.C) {
	// Make a dummy charm dir with revision 123.
	dir := charmtesting.Charms.ClonedDir(c.MkDir(), "dummy")
	dir.SetDiskRevision(123)
	// Now bundle the dir.
	tempFile, err := ioutil.TempFile(c.MkDir(), "charm")
	c.Assert(err, gc.IsNil)
	filename := tempFile.Name()
	defer tempFile.Close()
	defer os.Remove(filename)
	err = dir.BundleTo(tempFile)
	c.Assert(err, gc.IsNil)
	_, err = tempFile.Seek(0, os.SEEK_SET)
	c.Assert(err, gc.IsNil)

	// Now try uploading it and ensure the revision persists.
	resp := s.uploadBundle(c, filename, "quantal")
	expectedURL := charm.MustParseURL("local:quantal/dummy-123")
	s.checkUploadResponse(c, resp, expectedURL)
	sch, err := s.State.Charm(expectedURL)
	c.Assert(err, gc.IsNil)
	c.Assert(sch.URL(), gc.DeepEquals, expectedURL)
	c.Assert(sch.Revision(), gc.Equals, 123)
	c.Assert(sch.IsUploaded(), jc.IsTrue)
	// Finally, verify the SHA256 and uploaded URL.
	expectedSHA256, _, err := utils.ReadSHA256(tempFile)
	c.Assert(err, gc.IsNil)
	name := charm.Quote(expectedURL.String())
	storage, err := environs.GetStorage(s.State)
	c.Assert(err, gc.IsNil)
	expectedUploadURL, err := storage.URL(name)
	c.Assert(err, gc.IsNil)

	c.Assert(sch.BundleURL().String(), gc.Equals, expectedUploadURL)
	c.Assert(sch.BundleSha256(), gc.Equals, expectedSHA256)

	reader, err := storage.Get(name)
	c.Assert(err, gc.IsNil)
	defer reader.Close()
	downloadedSHA256, _, err := utils.ReadSHA256(reader)
	c.Assert(err, gc.IsNil)
	c.Assert(downloadedSHA256, gc.Equals, expectedSHA256)
}

func (s *charmsUploadSuite) TestCharmsUploadRepackagesNestedArchives(c *gc.C) {
	// Make a clone of the dummy charm in a nested directory.
	rootDir := c.MkDir()
	dirPath := filepath.Join(rootDir, "subdir1", "subdir2")
	err := os.MkdirAll(dirPath, 0755)
	c.Assert(err, gc.IsNil)
	dir := charmtesting.Charms.ClonedDir(dirPath, "dummy")
	// Now tweak the path the dir thinks it is in and bundle it.
	dir.Path = rootDir
	tempFile, err := ioutil.TempFile(c.MkDir(), "charm")
	c.Assert(err, gc.IsNil)
	filename := tempFile.Name()
	defer tempFile.Close()
	defer os.Remove(filename)
	err = dir.BundleTo(tempFile)
	c.Assert(err, gc.IsNil)

	// Try reading it as a bundle - should fail due to nested dirs.
	_, err = charm.ReadBundle(filename)
	c.Assert(err, gc.ErrorMatches, "bundle file not found: metadata.yaml")

	// Now try uploading it - should succeeed and be repackaged.
	resp := s.uploadBundle(c, filename, "quantal")
	expectedURL := charm.MustParseURL("local:quantal/dummy-1")
	s.checkUploadResponse(c, resp, expectedURL)
	sch, err := s.State.Charm(expectedURL)
	c.Assert(err, gc.IsNil)
	c.Assert(sch.URL(), gc.DeepEquals, expectedURL)
	c.Assert(sch.Revision(), gc.Equals, 1)
	c.Assert(sch.IsUploaded(), jc.IsTrue)

	// Get it from the storage and try to read it as a bundle - it
	// should succeed, because it was repackaged during upload to
	// strip nested dirs.
	archiveName := strings.TrimPrefix(sch.BundleURL().RequestURI(), "/dummyenv/private/")
	storage, err := environs.GetStorage(s.State)
	c.Assert(err, gc.IsNil)
	reader, err := storage.Get(archiveName)
	c.Assert(err, gc.IsNil)
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	c.Assert(err, gc.IsNil)
	downloadedFile, err := ioutil.TempFile(c.MkDir(), "downloaded")
	c.Assert(err, gc.IsNil)
	defer downloadedFile.Close()
	defer os.Remove(downloadedFile.Name())
	err = ioutil.WriteFile(downloadedFile.Name(), data, 0644)
	c.Assert(err, gc.IsNil)

	bundle, err := charm.ReadBundle(downloadedFile.Name())
	c.Assert(err, gc.IsNil)
	c.Assert(bundle.Revision(), jc.DeepEquals, sch.Revision())
	c.Assert(bundle.Meta(), jc.DeepEquals, sch.Meta())
	c.Assert(bundle.Config(), jc.DeepEquals, sch.Config())
}

//---------------------------
// charms get

type charmsGetSuite struct {
	charmsSuite
}

var _ = gc.Suite(&charmsGetSuite{})

func (s *charmsGetSuite) SetUpSuite(c *gc.C) {
	s.charmsSuite.SetUpSuite(c)
	s.httpMethod = "GET"
	s.dataMimetype = "application/zip"
}

func (s *charmsGetSuite) checkListResponse(
	c *gc.C, resp *http.Response, expFiles []string,
) bool {
	if !s.checkPossibleErrorResponse(c, resp) {
		return false
	}
	res := s.checkResponse(c, resp, http.StatusOK, "application/json")
	var result params.CharmsResponse
	s.jsonResponse(c, resp, &result)
	res = c.Check(result.Error, gc.Equals, "") && res
	return c.Check(result.Files, gc.DeepEquals, expFiles) && res
}

func (s *charmsGetSuite) TestCharmsGetRequiresCharmURL(c *gc.C) {
	resp := s.queryRequest(c, "file=hooks/install")
	s.checkErrorResponse(c, resp, http.StatusBadRequest,
		"expected url=CharmURL query argument")
}

func (s *charmsGetSuite) TestCharmsGetFailsWithInvalidCharmURL(c *gc.C) {
	resp := s.queryRequest(c, "url=local:precise/no-such")
	s.checkErrorResponse(c, resp, http.StatusBadRequest,
		"unable to retrieve and save the charm: charm not found in the provider storage: .*")
}

func (s *charmsGetSuite) TestCharmsGetReturnsNotFoundWhenMissing(c *gc.C) {
	// Add the dummy charm.
	ch := charmtesting.Charms.Bundle(c.MkDir(), "dummy")
	s.uploadBundle(c, ch.Path, "quantal")

	// Ensure a 404 is returned for files not included in the charm.
	for i, file := range []string{
		"no-such-file", "..", "../../../etc/passwd", "hooks/delete",
	} {
		c.Logf("test %d: %s", i, file)
		resp := s.queryRequest(c, "url=local:quantal/dummy-1&file="+file)
		c.Assert(resp.StatusCode, gc.Equals, http.StatusNotFound)
	}
}

func (s *charmsGetSuite) TestCharmsGetReturnsForbiddenWithDirectory(c *gc.C) {
	// Add the dummy charm.
	ch := charmtesting.Charms.Bundle(c.MkDir(), "dummy")
	s.uploadBundle(c, ch.Path, "quantal")

	// Ensure a 403 is returned if the requested file is a directory.
	resp := s.queryRequest(c, "url=local:quantal/dummy-1&file=hooks")
	c.Assert(resp.StatusCode, gc.Equals, http.StatusForbidden)
}

func (s *charmsGetSuite) TestCharmsGetReturnsFileContents(c *gc.C) {
	// Add the dummy charm.
	ch := charmtesting.Charms.Bundle(c.MkDir(), "dummy")
	s.uploadBundle(c, ch.Path, "quantal")

	// Ensure the file contents are properly returned.
	for i, t := range []struct {
		summary  string
		file     string
		response string
	}{{
		summary:  "relative path",
		file:     "revision",
		response: "1",
	}, {
		summary:  "exotic path",
		file:     "./hooks/../revision",
		response: "1",
	}, {
		summary:  "sub-directory path",
		file:     "hooks/install",
		response: "#!/bin/bash\necho \"Done!\"\n",
	},
	} {
		c.Logf("test %d: %s", i, t.summary)
		resp := s.queryRequest(c, "url=local:quantal/dummy-1&file="+t.file)
		s.checkRawResult(c, resp, t.response, "text/plain; charset=utf-8")
	}
}

func (s *charmsGetSuite) TestCharmsGetReturnsManifest(c *gc.C) {
	// Add the dummy charm.
	ch := charmtesting.Charms.Bundle(c.MkDir(), "dummy")
	s.uploadBundle(c, ch.Path, "quantal")

	// Ensure charm files are properly listed.
	resp := s.queryRequest(c, "url=local:quantal/dummy-1")
	s.checkResponse(c, resp, http.StatusOK, "application/json")
	manifest, err := ch.Manifest()
	c.Assert(err, gc.IsNil)
	expectedFiles := manifest.SortedValues()
	s.checkListResponse(c, resp, expectedFiles)
}

func (s *charmsGetSuite) TestCharmsGetUsesCache(c *gc.C) {
	// Add a fake charm archive in the cache directory.
	cacheDir := filepath.Join(s.DataDir(), "charm-get-cache")
	err := os.MkdirAll(cacheDir, 0755)
	c.Assert(err, gc.IsNil)

	// Create and save a bundle in it.
	charmDir := charmtesting.Charms.ClonedDir(c.MkDir(), "dummy")
	testPath := filepath.Join(charmDir.Path, "utils.js")
	contents := "// blah blah"
	err = ioutil.WriteFile(testPath, []byte(contents), 0755)
	c.Assert(err, gc.IsNil)
	var buffer bytes.Buffer
	err = charmDir.BundleTo(&buffer)
	c.Assert(err, gc.IsNil)
	charmArchivePath := filepath.Join(
		cacheDir, charm.Quote("local:trusty/django-42")+".zip")
	err = ioutil.WriteFile(charmArchivePath, buffer.Bytes(), 0644)
	c.Assert(err, gc.IsNil)

	// Ensure the cached contents are properly retrieved.
	resp := s.queryRequest(c, "url=local:trusty/django-42&file=utils.js")
	s.checkRawResult(c, resp, contents, "application/javascript")
}
