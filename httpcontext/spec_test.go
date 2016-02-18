// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext_test

import (
	"net/url"

	"github.com/juju/persistent-cookiejar"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/httpcontext"
)

type SpecSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&SpecSuite{})

func (s *SpecSuite) TestNewSpecOkay(c *gc.C) {
	spec := httpcontext.NewSpec()

	c.Check(spec.CookieJarOptions, jc.DeepEquals, &cookiejar.Options{
		Filename: cookiejar.DefaultCookieFile(),
	})
	c.Check(spec.CharmStoreURL, gc.IsNil)
	// We can't check spec.VisitWebPage (Go disallows func comparison).
	c.Check(spec.VisitWebPage, gc.NotNil)
}

func (s *SpecSuite) TestNewSpecEnvVar(c *gc.C) {
	s.PatchEnvironment("JUJU_COOKIEFILE", "some/file")

	spec := httpcontext.NewSpec()

	c.Check(spec.CookieJarOptions, jc.DeepEquals, &cookiejar.Options{
		Filename: "some/file",
	})
}

func (s *SpecSuite) TestNewContext(c *gc.C) {
	stub := &testing.Stub{}
	deps := &stubSpecDeps{Stub: stub}
	csURL, err := url.Parse("cs:trusty/spam")
	c.Assert(err, jc.ErrorIsNil)
	spec := httpcontext.Spec{
		Deps: deps,
		CookieJarOptions: &cookiejar.Options{
			Filename: "some/file",
		},
		VisitWebPage:  nil,
		CharmStoreURL: csURL,
	}

	ctx, err := spec.NewContext()
	c.Assert(err, jc.ErrorIsNil)

	stub.CheckCallNames(c, "NewCookieJar")
	stub.CheckCall(c, 0, "NewCookieJar", spec.CookieJarOptions)
	// TODO(ericsnow) How to test the context?
	c.Check(ctx, gc.NotNil)
}

type stubSpecDeps struct {
	httpcontext.ContextDeps
	*testing.Stub

	ReturnNewCookieJar httpcontext.CookieJar
}

func (s *stubSpecDeps) NewCookieJar(opts *cookiejar.Options) (httpcontext.CookieJar, error) {
	s.AddCall("NewCookieJar", opts)
	if err := s.NextErr(); err != nil {
		return nil, err
	}

	return s.ReturnNewCookieJar, nil
}
