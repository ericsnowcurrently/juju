// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext_test

import (
	"net/http"
	"net/url"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charmrepo.v2-unstable/csclient"
	"gopkg.in/macaroon-bakery.v1/httpbakery"

	"github.com/juju/juju/charmstore"
	"github.com/juju/juju/httpcontext"
)

type ContextSuite struct {
	testing.IsolationSuite

	stub *testing.Stub
	jar  *stubCookieJar
	deps *stubContextDeps
}

var _ = gc.Suite(&ContextSuite{})

func (s *ContextSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)

	s.stub = &testing.Stub{}
	s.jar = &stubCookieJar{Stub: s.stub}
	s.deps = &stubContextDeps{Stub: s.stub}
}

func (s *ContextSuite) newContext(c *gc.C) *httpcontext.Context {
	spec := httpcontext.Spec{
		Deps: &stubSpecDeps{
			ContextDeps:        s.deps,
			Stub:               s.stub,
			ReturnNewCookieJar: s.jar,
		},
	}
	ctx, err := spec.NewContext()
	c.Assert(err, jc.ErrorIsNil)
	s.stub.ResetCalls()
	return ctx
}

func (s *ContextSuite) TestClose(c *gc.C) {
	ctx := s.newContext(c)

	err := ctx.Close()
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCallNames(c, "Save")
}

func (s *ContextSuite) TestConnect(c *gc.C) {
	expected := &http.Client{}
	s.deps.ReturnNewHTTPClient = expected
	ctx := s.newContext(c)

	client := ctx.Connect()

	s.stub.CheckCallNames(c, "NewHTTPClient")
	c.Check(client.Jar, gc.Equals, s.jar)
	client.Jar = expected.Jar
	c.Check(client, gc.Equals, expected)
}

func (s *ContextSuite) TestConnectToBakery(c *gc.C) {
	expected := &httpbakery.Client{}
	s.deps.ReturnNewBakeryClient = expected
	ctx := s.newContext(c)

	client := ctx.ConnectToBakery()

	s.stub.CheckCallNames(c, "NewBakeryClient")
	// We can't check an actual visit func since Go disallows comparing functions.
	s.stub.CheckCall(c, 0, "NewBakeryClient", s.jar, (func(*url.URL) error)(nil))
	c.Check(client, gc.Equals, expected)
}

func (s *ContextSuite) TestConnectToCharmStore(c *gc.C) {
	httpClient := &http.Client{}
	s.deps.ReturnNewHTTPClient = httpClient
	expected := &stubCharmStoreClient{Stub: s.stub}
	s.deps.ReturnNewCharmStoreClient = expected
	ctx := s.newContext(c)

	client := ctx.ConnectToCharmStore()

	s.stub.CheckCallNames(c, "NewHTTPClient", "NewCharmStoreClient")
	s.stub.CheckCall(c, 1, "NewCharmStoreClient", csclient.Params{
		URL:          "",
		HTTPClient:   httpClient,
		VisitWebPage: nil,
	})
	c.Check(client, gc.Equals, expected)
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

type stubContextDeps struct {
	*testing.Stub

	ReturnNewHTTPClient       *http.Client
	ReturnNewBakeryClient     httpcontext.BakeryClient
	ReturnNewCharmStoreClient charmstore.Client
}

func (s *stubContextDeps) NewHTTPClient() *http.Client {
	s.AddCall("NewHTTPClient")
	s.NextErr() // Pop one off.

	return s.ReturnNewHTTPClient
}

func (s *stubContextDeps) NewBakeryClient(jar httpcontext.CookieJar, visit func(*url.URL) error) httpcontext.BakeryClient {
	s.AddCall("NewBakeryClient", jar, visit)
	s.NextErr() // Pop one off.

	return s.ReturnNewBakeryClient
}

func (s *stubContextDeps) NewCharmStoreClient(args csclient.Params) charmstore.Client {
	s.AddCall("NewCharmStoreClient", args)
	s.NextErr() // Pop one off.

	return s.ReturnNewCharmStoreClient
}
