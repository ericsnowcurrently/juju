// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext_test

import (
	"net/http"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charmrepo.v2-unstable/csclient"
	"gopkg.in/macaroon-bakery.v1/httpbakery"

	"github.com/juju/juju/httpcontext"
)

type ContextSuite struct {
	testing.IsolationSuite

	stub *testing.Stub
	jar  *stubCookieJar
}

var _ = gc.Suite(&ContextSuite{})

func (s *ContextSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)

	s.stub = &testing.Stub{}
	s.jar = &stubCookieJar{Stub: s.stub}
}

func (s *ContextSuite) TestClose(c *gc.C) {
	ctx := httpcontext.NewContext(s.jar)

	err := ctx.Close()
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCallNames(c, "Save")
}

func (s *ContextSuite) TestConnect(c *gc.C) {
	ctx := httpcontext.NewContext(s.jar)

	client := ctx.Connect()

	s.stub.CheckNoCalls(c)
	c.Check(client.Jar, gc.Equals, s.jar)
}

func (s *ContextSuite) TestConnectToBakery(c *gc.C) {
	ctx := httpcontext.NewContext(s.jar)

	client := ctx.ConnectToBakery()

	s.stub.CheckNoCalls(c)
	c.Check(client.(*httpbakery.Client).Jar, gc.Equals, s.jar)
	// We can't check an actual func since Go disallows comparing functions.
	c.Check(client.(*httpbakery.Client).VisitWebPage, gc.IsNil)
}

func (s *ContextSuite) TestConnectToCharmStore(c *gc.C) {
	ctx := httpcontext.NewContext(s.jar)

	client := ctx.ConnectToCharmStore()

	s.stub.CheckNoCalls(c)
	c.Check(client.ServerURL(), gc.Equals, csclient.ServerURL)
}

type stubCookieJar struct {
	http.CookieJar
	*testing.Stub
}

func (s *stubCookieJar) Save() error {
	s.AddCall("Save")
	if err := s.NextErr(); err != nil {
		return err
	}

	return nil
}
