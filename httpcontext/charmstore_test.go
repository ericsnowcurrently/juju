// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext_test

import (
	"net/url"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/macaroon.v1"

	"github.com/juju/juju/httpcontext"
)

type CharmStoreSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&CharmStoreSuite{})

func (s *CharmStoreSuite) TestNewCharmStoreClient(c *gc.C) {
	auth, err := macaroon.New(nil, "spam", "<loc>")
	c.Assert(err, jc.ErrorIsNil)

	client, err := httpcontext.NewCharmStoreClient(auth)
	c.Assert(err, jc.ErrorIsNil)

	// TODO(ericsnow) How to test the client?
	c.Check(client, gc.NotNil)
}

func (s *CharmStoreSuite) TestAuthorize(c *gc.C) {
	expected, err := macaroon.New(nil, "spam", "<loc>")
	c.Assert(err, jc.ErrorIsNil)
	stub := &testing.Stub{}
	client := &stubCharmStoreClient{Stub: stub}
	client.ReturnGet = expected
	id := charm.MustParseURL("cs:trusty/spam")

	auth, err := httpcontext.AuthorizeForCharm(client, id)
	c.Assert(err, jc.ErrorIsNil)

	stub.CheckCallNames(c, "Get")
	stub.CheckCall(c, 0, "Get", "/delegatable-macaroon?id="+url.QueryEscape(id.String()), &expected)
	c.Check(auth, jc.DeepEquals, expected)
}

type stubCharmStoreClient struct {
	*testing.Stub

	ReturnGet *macaroon.Macaroon
}

func (s *stubCharmStoreClient) Get(path string, result interface{}) error {
	s.AddCall("Get", path, result)
	if err := s.NextErr(); err != nil {
		return err
	}

	resVal := result.(**macaroon.Macaroon)
	*resVal = s.ReturnGet

	return nil
}
