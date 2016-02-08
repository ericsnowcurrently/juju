// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cmd

import (
	"net/http"

	"github.com/juju/errors"
	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/juju/charmrepo.v2-unstable"
	"gopkg.in/juju/charmrepo.v2-unstable/csclient"
	"gopkg.in/macaroon-bakery.v1/httpbakery"
	"gopkg.in/macaroon.v1"
)

// CharmStoreClient gives access to the charm store server and provides
// parameters for connecting to the charm store.
type CharmStoreClient struct {
	params charmrepo.NewCharmStoreParams
}

// NewCharmStoreClient is called to obtain a charm store client
// including the parameters for connecting to the charm store, and
// helpers to save the local authorization cookies and to authorize
// non-public charm deployments.
func NewCharmStoreClient(client *http.Client) *CharmStoreClient {
	return &CharmStoreClient{
		params: charmrepo.NewCharmStoreParams{
			HTTPClient:   client,
			VisitWebPage: httpbakery.OpenWebBrowser,
		},
	}
}

// Authorize acquires and returns the charm store delegatable macaroon
// to be used to add the charm corresponding to the given URL. The
// macaroon is properly attenuated so that it can only be used to deploy
// the given charm URL.
func (client *CharmStoreClient) Authorize(curl *charm.URL) (*macaroon.Macaroon, error) {
	if curl == nil {
		return nil, errors.New("empty charm url not allowed")
	}

	client := csclient.New(csclient.Params{
		URL:          c.params.URL,
		HTTPClient:   c.params.HTTPClient,
		VisitWebPage: c.params.VisitWebPage,
	})
	endpoint := "/delegatable-macaroon"
	endpoint += "?id=" + url.QueryEscape(curl.String())

	var m *macaroon.Macaroon
	if err := client.Get(endpoint, &m); err != nil {
		return nil, errors.Trace(err)
	}

	// We need to add the is-entity first party caveat to the
	// delegatable macaroon in case we're talking to the old
	// version of the charmstore.
	// TODO (ashipika) - remove this once the new charmstore
	// is deployed.
	if err := m.AddFirstPartyCaveat("is-entity " + curl.String()); err != nil {
		return nil, errors.Trace(err)
	}

	return m, nil
}
