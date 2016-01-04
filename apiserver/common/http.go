// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common

import (
	"net/http"
	"strings"
)

// HTTPHandlerMux exposes methods that register a handler for a URL
// path pattern relative to a specific HTTP method.
// (See "github.com/bmizerany/pat".)
type HTTPHandlerMux interface {
	Get(string, http.Handler)
	Post(string, http.Handler)
	Head(string, http.Handler)
	Put(string, http.Handler)
	Del(string, http.Handler)
	Options(string, http.Handler)
}

// HTTPEndpoint describes a single HTTP endpoint.
type HTTPEndpoint struct {
	// Methods is the list of supported HTTP methods.
	Methods []string

	// Pattern is the URL path pattern corresponding to the endpoint.
	// (See "github.com/bmizerany/pat".)
	Pattern string

	// Handler the HTTP handler that will handle requests
	// on the endpoint.
	Handler http.Handler
}

// Register associates the endpoints pattern with its handler
// on the given mux.
func (ep HTTPEndpoint) Register(mux HTTPHandlerMux) {
	methods := ep.Methods
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "DEL", "HEAD", "OPTIONS"}
	}

	for _, method := range methods {
		method = strings.ToUpper(method)
		switch method {
		case "GET":
			mux.Get(ep.Pattern, ep.Handler)
		case "POST":
			mux.Post(ep.Pattern, ep.Handler)
		case "PUT":
			mux.Put(ep.Pattern, ep.Handler)
		case "DEL":
			mux.Del(ep.Pattern, ep.Handler)
		case "HEAD":
			mux.Head(ep.Pattern, ep.Handler)
		case "OPTIONS":
			mux.Options(ep.Pattern, ep.Handler)
		default:
			// TODO(ericsnow) fail?
		}
	}
}
