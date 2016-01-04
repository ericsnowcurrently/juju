// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common

import (
	"net/http"
	"strings"
)

// NewHTTPHandlerArgs holds the args to the func in the NewHTTPHandler
// field of HTTPHandlerSpec.
type NewHTTPHandlerArgs struct {
	// DataDir is the top-level data directory to use.
	DataDir string

	// LogDir is the top-level log directory to use.
	LogDir string

	// Stop is closed when the server is about to be killed.
	Stop <-chan struct{}

	// Connect is the function that is used to connect to Juju's state
	// for the given HTTP request.
	Connect func(*http.Request) (*state.State, error)
}

// HTTPHandlerSpec defines an HTTP handler for a specific endpoint
// on Juju's HTTP server. Such endpoints facilitate behavior that is
// not supported through normal (websocket) RPC. That includes file
// transfer.
type HTTPHandlerSpec struct {
	// MethodHandlers associates each supported HTTP method
	// with a handler factory.
	MethodHandlers map[string]func(NewHTTPHandlerArgs) http.Handler

	// AuthKind defines the kind of authenticated "user" that the
	// handler supports. This correlates directly to entities, as
	// identified by tag kinds (e.g. names.UserTagKind). The empty
	// string indicates support for unauthenticated requests.
	AuthKind string

	// StrictValidation is the value that will be used for the handler's
	// httpContext (see apiserver/httpcontext.go).
	StrictValidation bool

	// StateServerEnvOnly is the value that will be used for the handler's
	// httpContext (see apiserver/httpcontext.go).
	StateServerEnvOnly bool
}

func NewHTTPHandlerSpec(authKind string, newHandler func(NewHTTPHandlerArgs) http.Handler, methods ...string) HTTPHandlerSpec {
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "DEL", "HEAD", "OPTIONS"}
	}

	spec := HTTPHandlerSpec{
		MethodHandlers: make(map[string]func(NewHTTPHandlerArgs) http.Handler),
		AuthKind:       authKind,
	}
	for _, method := range methods {
		spec.MethodHandlers[method] = newHandler
	}
	return spec
}

type HTTPEndpointSpec struct {
	// Pattern is the "mux" (URL path) pattern for the endpoint.
	Pattern string

	// Handler defines the handler for the endpoint.
	Handler HTTPHandlerSpec
}

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
	// Pattern is the URL path pattern corresponding to the endpoint.
	// (See "github.com/bmizerany/pat".)
	Pattern string

	// MethodHandlers associates each supported HTTP method
	// with an HTTP handler.
	MethodHandlers map[string]http.Handler
}

func NewHTTPEndpoint(pattern string, handler http.Handler, methods ...string) HTTPEndpoint {
	return HTTPEndpoint{
		Methods: methods,
		Pattern: path.Clean(path.Join("/", pattern)),
		Handler: handler,
	}
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
