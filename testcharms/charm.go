// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package testcharms holds a corpus of charms
// for testing.
package testcharms

import (
	"os"
	"path/filepath"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"
	charmresource "gopkg.in/juju/charm.v6-unstable/resource"
	"gopkg.in/juju/charmrepo.v2-unstable/csclient"
	"gopkg.in/juju/charmrepo.v2-unstable/csclient/params"
	"gopkg.in/juju/charmrepo.v2-unstable/testing"

	"github.com/juju/juju/resource"
)

// Repo provides access to the test charm repository.
var Repo = testing.NewRepo("charm-repo", "quantal")

// UploadCharm uploads a charm using the given charm store client, and returns
// the resulting charm URL and charm.
func UploadCharm(c *gc.C, client *csclient.Client, url, name string) (*charm.URL, charm.Charm) {
	id := charm.MustParseURL(url)
	promulgatedRevision := -1
	if id.User == "" {
		// We still need a user even if we are uploading a promulgated charm.
		id.User = "who"
		promulgatedRevision = id.Revision
	}
	ch := Repo.CharmArchive(c.MkDir(), name)

	// Upload the charm.
	err := client.UploadCharmWithRevision(id, ch, promulgatedRevision)
	c.Assert(err, jc.ErrorIsNil)

	// Allow read permissions to everyone.
	err = client.Put("/"+id.Path()+"/meta/perm/read", []string{params.Everyone})
	c.Assert(err, jc.ErrorIsNil)

	// Return the charm and its URL.
	return id, ch
}

// UploadCharmMultiSeries uploads a charm with revision using the given charm store client,
// and returns the resulting charm URL and charm. This API caters for new multi-series charms
// which do not specify a series in the URL.
func UploadCharmMultiSeries(c *gc.C, client *csclient.Client, url, name string) (*charm.URL, charm.Charm) {
	id := charm.MustParseURL(url)
	if id.User == "" {
		// We still need a user even if we are uploading a promulgated charm.
		id.User = "who"
	}
	ch := Repo.CharmArchive(c.MkDir(), name)

	// Upload the charm.
	curl, err := client.UploadCharm(id, ch)
	c.Assert(err, jc.ErrorIsNil)

	// Allow read permissions to everyone.
	err = client.Put("/"+curl.Path()+"/meta/perm/read", []string{params.Everyone})
	c.Assert(err, jc.ErrorIsNil)

	// Return the charm and its URL.
	return curl, ch
}

// UploadBundle uploads a bundle using the given charm store client, and
// returns the resulting bundle URL and bundle.
func UploadBundle(c *gc.C, client *csclient.Client, url, name string) (*charm.URL, charm.Bundle) {
	id := charm.MustParseURL(url)
	promulgatedRevision := -1
	if id.User == "" {
		// We still need a user even if we are uploading a promulgated bundle.
		id.User = "who"
		promulgatedRevision = id.Revision
	}
	b := Repo.BundleArchive(c.MkDir(), name)

	// Upload the bundle.
	err := client.UploadBundleWithRevision(id, b, promulgatedRevision)
	c.Assert(err, jc.ErrorIsNil)

	// Allow read permissions to everyone.
	err = client.Put("/"+id.Path()+"/meta/perm/read", []string{params.Everyone})
	c.Assert(err, jc.ErrorIsNil)

	// Return the bundle and its URL.
	return id, b
}

// ExtractResourceInfo gathers up the resources in the provided charm
// dir. Only the resource info is returned, though the size and
// fingerprint of the resource files are included. The origin and
// revision default to "store" and 0, respectively.
//
// The files must be in the charm dir and match the filename
// in the charm metadata.
func ExtractResourceInfo(c *gc.C, ch *charm.CharmDir) []charmresource.Resource {
	var resources []charmresource.Resource
	for _, opened := range ExtractResources(c, ch) {
		opened.Close()
		resources = append(resources, opened.Resource.Resource)
	}
	return resources
}

// TODO(ericsnow) Return []charmresource.Opened once it exists.

// ExtractResources gathers up the resources in the provided charm dir.
// This includes both the full resource info and the resource files
// themselves. The origin and revision default to "store" and 0,
// respectively.
//
// The files must be in the charm dir and match the filename
// in the charm metadata.
func ExtractResources(c *gc.C, ch *charm.CharmDir) []resource.Opened {
	var resources []resource.Opened
	for _, meta := range ch.Meta().Resources {
		resFile, chRes := openResource(c, ch.Path, meta)
		resources = append(resources, resource.Opened{
			Resource:   resource.Resource{Resource: chRes},
			ReadCloser: resFile,
		})
	}
	return resources
}

func openResource(c *gc.C, rootDir string, meta charmresource.Meta) (*os.File, charmresource.Resource) {
	resFile, err := os.Open(filepath.Join(rootDir, meta.Path))
	c.Assert(err, jc.ErrorIsNil)

	finfo, err := resFile.Stat()
	c.Assert(err, jc.ErrorIsNil)
	fp, err := charmresource.GenerateFingerprint(resFile)
	c.Assert(err, jc.ErrorIsNil)
	chRes := charmresource.Resource{
		Meta:        meta,
		Origin:      charmresource.OriginStore,
		Revision:    0,
		Fingerprint: fp,
		Size:        finfo.Size(),
	}

	_, err = resFile.Seek(0, os.SEEK_SET)
	c.Assert(err, jc.ErrorIsNil)
	return resFile, chRes
}
