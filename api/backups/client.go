// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"net/http"
	"strings"

	"github.com/juju/juju/api/base"
)

type httpClient interface {
	SendHTTPRequest(method, path string, args interface{}) (*http.Request, *http.Response, error)
}

type apiState interface {
	base.APICallCloser
	httpClient

	// Addr returns the address used to connect to the API server.
	Addr() string
}

// Client wraps the backups API for the client.
type Client struct {
	base.ClientFacade
	facade        base.FacadeCaller
	http          httpClient
	baseFacade    base.FacadeCaller
	publicAddress string
}

// NewClient returns a new backups API client.
func NewClient(st apiState) *Client {
	publicAddress := strings.SplitN(st.Addr(), ":", 2)[0]
	frontend, backend := base.NewClientFacade(st, "Backups")
	return &Client{
		ClientFacade:  frontend,
		facade:        backend,
		http:          st,
		publicAddress: publicAddress,
	}
}
