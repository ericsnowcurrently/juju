// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package simple

import (
	"io"
	"net/http"
)

type HTTPPayload struct {
	Data     io.Reader
	Mimetype string
}

func (p *HTTPPayload) AddHeader(req *http.Request) {
	mimetype := p.Mimetype
	if mimetype == "" {
		mimetype = "application/octet-stream"
	}
	req.Header.Set("Content-Type", mimetype)
}
