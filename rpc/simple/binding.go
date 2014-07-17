// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package simple

import (
	"fmt"
	"net/url"
)

type APIBinding struct {
	Name       string
	HTTPMethod string
	Mimetype   string
	// XXX Add Handler?
}

type APIHost struct {
	Scheme   string
	Hostname string
	RootPath string
}

func NewAPIHost(hostname string) *APIHost {
	host := APIHost{
		Scheme:   "https",
		Hostname: hostname,
		RootPath: "",
	}
	return &host
}

func (ah *APIHost) URL(uuid, apimethod, query string) *url.URL {
	if query != "" && query[0] == '?' {
		query = query[1:]
	}
	rootpath := ah.RootPath
	if rootpath == "" {
		if uuid == "" {
			rootpath = "/" // legacy API
		} else {
			rootpath = fmt.Sprintf("/environment/%s/", uuid)
		}
	}
	URL := url.URL{
		Scheme:   ah.Scheme,
		Host:     ah.Hostname,
		Path:     rootpath + apimethod,
		RawQuery: query,
	}
	return &URL
}
