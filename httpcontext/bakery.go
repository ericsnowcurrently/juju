// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext

import (
	"io"
	"net/http"
	"net/url"

	"gopkg.in/macaroon-bakery.v1/httpbakery"
	"gopkg.in/macaroon.v1"
)

// BakeryClient provides the functionality of httpbakery.Client
// as used in Juju.
type BakeryClient interface {
	// TODO(ericsnow) Return a BakeryHTTPClient from a HTTPClient()
	// method instead of embedding?
	bakeryHTTPClient
	httpbakery.DischargeAcquirer

	// DischargeAll attempts to acquire discharge macaroons for all
	// the third party caveats in m, and returns a slice containing
	// all of them bound to m.
	//
	// Note that the returned macaroon slice is not stored in the
	// client cookie jar.
	//
	// See httpbakery.Client for more details.
	DischargeAll(m *macaroon.Macaroon) (macaroon.Slice, error)
}

type bakeryHTTPClient interface {
	// Do sends the given HTTP request and returns its response.
	// Macaroon discharges are handled appopriately. This may include
	// opening a web browser to handle user interaction.
	//
	// See httpbakery.Client for more details.
	Do(req *http.Request) (*http.Response, error)

	// DoWithBody is like Do except that the given body is used for the
	// body of the HTTP request, and reset to its start by seeking if
	// the request is retried. It is an error if req.Body is non-zero.
	//
	// Note that, unlike the request body passed to http.NewRequest,
	// the body will not be closed even if implements io.Closer.
	//
	// See httpbakery.Client for more details.
	DoWithBody(req *http.Request, body io.ReadSeeker) (*http.Response, error)

	// DoWithBodyAndCustomError is like DoWithBody except it allows
	// a client to specify a custom error function, getError, which
	// is called on the HTTP response and may return a non-nil error
	// if the response holds an error. If the cause of the returned
	// error is a *Error value and its code is ErrDischargeRequired,
	// the macaroon in its Info field will be discharged and the
	// request will be repeated with the discharged macaroon. If
	// getError returns nil, it should leave the response body
	// unchanged.
	//
	// See httpbakery.Client for more details.
	DoWithBodyAndCustomError(req *http.Request, body io.ReadSeeker, getError func(resp *http.Response) error) (*http.Response, error)

	// HandleError tries to resolve the given error, which should be a
	// response to the given URL, by discharging any macaroon contained
	// in it.
	//
	// See httpbakery.Client for more details.
	HandleError(reqURL *url.URL, err error) error
}
