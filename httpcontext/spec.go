// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext

import (
	"net/url"
	"os"

	"github.com/juju/errors"
	"github.com/juju/persistent-cookiejar"
	"gopkg.in/macaroon-bakery.v1/httpbakery"
)

// Spec holds the information needed for HTTP connections in Juju that
// rely on macaroons. This includes connections to the Juju API and to
// the charm store.
type Spec struct {
	// Deps is the spec's external dependencies.
	Deps SpecDeps

	// CookieJarOptions holds the arguments for cookiejar.New().
	CookieJarOptions *cookiejar.Options

	// VisitWebPage is used to visit a web page.
	VisitWebPage func(*url.URL) error

	// CharmStoreURL is the URL to the charm store.
	CharmStoreURL *url.URL
}

// NewSpec creates a new spec using the Juju defaults:
//
//  cookie jar filename: $JUJU_COOKIEFILE or $GOCOOKIES or $HOME/.go-cookies
//  visit web page:      open a web browser
//  charm store URL:     the official charm store
func NewSpec() Spec {
	options := &cookiejar.Options{
		Filename: cookieFile(),
	}

	spec := Spec{
		Deps:             &specDeps{},
		CookieJarOptions: options,
		VisitWebPage:     httpbakery.OpenWebBrowser,
	}
	return spec
}

// NewContext generates a new HTTP context based on the spec.
func (spec Spec) NewContext() (*Context, error) {
	jar, err := spec.Deps.NewCookieJar(spec.CookieJarOptions)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ctx := &Context{
		deps:         spec.Deps,
		jar:          jar,
		visitWebPage: spec.VisitWebPage,
		csURL:        spec.CharmStoreURL,
	}
	return ctx, nil
}

// SpecDeps exposes the external dependencies of Spec.
type SpecDeps interface {
	ContextDeps

	// NewCookieJar returns a new HTTP cookie jar for the options.
	NewCookieJar(*cookiejar.Options) (CookieJar, error)
}

type specDeps struct {
	contextDeps
}

// NewCookieJar implements SpecDeps.
func (specDeps) NewCookieJar(opts *cookiejar.Options) (CookieJar, error) {
	return cookiejar.New(opts)
}

// cookieFile returns the path to the cookie used to store authorization
// macaroons. The returned value can be overridden by setting the
// JUJU_COOKIEFILE or GO_COOKIEFILE environment variables.
func cookieFile() string {
	if file := os.Getenv("JUJU_COOKIEFILE"); file != "" {
		return file
	}
	return cookiejar.DefaultCookieFile()
}
