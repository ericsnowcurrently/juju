// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext

import (
	"net/http"
	"net/url"

	"github.com/juju/errors"
	"gopkg.in/macaroon-bakery.v1/httpbakery"
	"gopkg.in/macaroon.v1"

	"github.com/juju/juju/charmstore"
)

// Based on:
// - apiserver/service/charmstore.go
// - cmd/juju/charmcmd/store.go
// - cmd/modelcmd/base.go
// - cmd/juju/service/store.go
// - cmd/juju/service/deploy.go

// CookieJar exposes the cookie jar functionality needed for Context.
type CookieJar interface {
	http.CookieJar

	// Save copies the jar into persistent storage.
	Save() error
}

// Context provides support for HTTP connections in Juju that use
// macaroons, particularly for authentication. It stores obtained
// macaroons and discharges in a cookie jar file.
type Context struct {
	jar          CookieJar
	visitWebPage func(*url.URL) error
	csURL        *url.URL
}

// NewContext returns a new Juju HTTP context that uses the provided
// cookie jar. The default charm store URL is used and the
// "visit web page" func is left unset. Use Spec.NewContext() if
// further customization is required.
func NewContext(jar CookieJar) *Context {
	return &Context{
		jar:          jar,
		visitWebPage: nil,
		csURL:        nil, // Use the default.
	}
}

func (ctx *Context) setAuth(url *url.URL, auth *macaroon.Macaroon) {
	// Set the provided authorizing macaroon as a cookie in the context.
	// TODO discharge any third party caveats in the macaroon.
	ms := []*macaroon.Macaroon{auth}
	httpbakery.SetCookie(ctx.jar, url, ms)

	// TODO(ericsnow) Are there other reasons to visit than auth?
	ctx.visitWebPage = nil
}

// Close saves the embedded cookie jar.
func (ctx *Context) Close() error {
	if err := ctx.jar.Save(); err != nil {
		return errors.Annotatef(err, "cannot save cookie jar")
	}
	return nil
}

// Connect returns an http.Client that contains the loaded
// persistent cookie jar.
func (ctx Context) Connect() *http.Client {
	client := httpbakery.NewHTTPClient()
	client.Jar = ctx.jar
	return client
}

// ConnectToBakery returns a new HTTP (macaroon) bakery client.
func (ctx Context) ConnectToBakery() BakeryClient {
	client := httpbakery.NewClient()
	client.Jar = ctx.jar
	client.VisitWebPage = ctx.visitWebPage
	return client
}

// ConnectToCharmStore returns a new charm store client.
func (ctx Context) ConnectToCharmStore() charmstore.Client {
	args := csClientArgs{
		URL: ctx.csURL,
	}
	args.HTTPClient = ctx.Connect()
	args.VisitWebPage = ctx.visitWebPage
	client := newCharmStoreClient(args)
	return client
}
