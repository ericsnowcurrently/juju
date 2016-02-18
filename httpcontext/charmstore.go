// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpcontext

// TODO(ericsnow) Move this file to a separate package?

import (
	"net/url"

	"github.com/juju/errors"
	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/juju/charmrepo.v2-unstable/csclient"
	"gopkg.in/macaroon.v1"

	"github.com/juju/juju/charmstore"
)

// NewCharmStoreClient returns a new charm store client for the given
// auth token. The default HTTP context spec is used.
func NewCharmStoreClient(auth *macaroon.Macaroon) (charmstore.Client, error) {
	spec := NewSpec()
	ctx, err := spec.NewContext()
	if err != nil {
		return nil, errors.Trace(err)
	}

	if auth != nil {
		csURL := ctx.csURL
		if csURL == nil {
			csURL, err = url.Parse(csclient.ServerURL)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
		ctx.setAuth(csURL, auth)
	}

	client := ctx.ConnectToCharmStore()
	return client, nil
}

type csClientArgs struct {
	csclient.Params
	// URL is the root endpoint URL of the charm store.
	URL *url.URL
}

func newCharmStoreClient(args csClientArgs) charmstore.Client {
	csArgs := args.Params // a copy
	if args.URL != nil {
		csArgs.URL = args.URL.String()
	}
	client := csclient.New(csArgs)

	return client
}

// AuthorizeForCharm acquires and returns the charm store delegatable macaroon
// to be used to add the charm corresponding to the given URL. The
// macaroon is properly attenuated so that it can only be used to deploy
// the given charm URL.
func AuthorizeForCharm(client charmstore.Client, curl *charm.URL) (*macaroon.Macaroon, error) {
	if curl == nil {
		return nil, errors.New("empty charm url not allowed")
	}

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
