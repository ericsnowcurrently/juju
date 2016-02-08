// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cmd

import (
	"net/http"
	"os"

	"github.com/juju/errors"
	"github.com/juju/persistent-cookiejar"
	"gopkg.in/macaroon-bakery.v1/httpbakery"
)

// APIContext is a convenience type that can be embedded wherever
// we need an API connection.
// It also stores a bakery bakery client allowing the API
// to be used using macaroons to authenticate. It stores
// obtained macaroons and discharges in a cookie jar file.
type APIContext struct {
	jar    *cookiejar.Jar
	client *httpbakery.Client
}

// NewAPIContext returns a new api context, which should be closed
// when done with.
func NewAPIContext() (*APIContext, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		Filename: cookieFile(),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	client := httpbakery.NewClient()
	client.Jar = jar
	client.VisitWebPage = httpbakery.OpenWebBrowser

	ctx := &APIContext{
		jar:    jar,
		client: client,
	}
	return ctx, nil
}

// Close saves the embedded cookie jar.
func (ctx *APIContext) Close() error {
	if err := ctx.jar.Save(); err != nil {
		return errors.Annotatef(err, "cannot save cookie jar")
	}
	return nil
}

// BakeryClient exposes the underlying HTTP client.
func (ctx APIContext) BakeryClient() *httpbakery.Client {
	return ctx.client
}

// HTTPClient returns an http.Client that contains the loaded
// persistent cookie jar.
func (ctx *APIContext) HTTPClient() *http.Client {
	return ctx.client.Client
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
