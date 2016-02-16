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
		CookieJarOptions: options,
		VisitWebPage:     httpbakery.OpenWebBrowser,
	}
	return spec
}

// NewContext generates a new HTTP context based on the spec.
func (spec Spec) NewContext() (*Context, error) {
	jar, err := cookiejar.New(spec.CookieJarOptions)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ctx := &Context{
		jar:          jar,
		visitWebPage: spec.VisitWebPage,
		csURL:        spec.CharmStoreURL,
	}
	return ctx, nil
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
