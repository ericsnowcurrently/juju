// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package simple

import (
	"net/http"
)

type HTTPBasicAuth struct {
	UserID   string
	Password string
}

func (a *HTTPBasicAuth) AddHeader(req *http.Request) {
	if a.UserID == "" && a.Password == "" {
		return
	}
	req.SetBasicAuth(a.UserID, a.Password)
}
