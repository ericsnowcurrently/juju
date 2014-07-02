// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

import (
	"fmt"
	"net/http"

	"github.com/juju/juju/state/apiserver/rawrpc"
)

// rawRPCHandler handles raw HTTP requests.
type rawRPCHandler struct {
	httpHandler
}

func (h *rawRPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	// Validate the request.
	if err := h.authenticate(r); err != nil {
		h.authError(w, h)
		return
	}
	if err := h.validateEnvironUUID(r); err != nil {
		h.sendError(w, http.StatusNotFound, err.Error())
		return
	}

	// Handle the request.
	wasHandled := false
	switch r.Method {
	case "POST":
		wasHandled = h.handlePOST(w, r)
	case "GET":
		wasHandled = h.handleGET(w, r)
	}
	if !wasHandled {
		// The method must be unsupported.
		h.sendError(w, http.StatusMethodNotAllowed, fmt.Sprintf("unsupported method: %q", r.Method))
	}
}

func (h *rawRPCHandler) handlePost(w http.ResponseWriter, r *http.Request) bool {
	return false
}

func (h *rawRPCHandler) handleGet(w http.ResponseWriter, r *http.Request) bool {
	return false
}

// sendError sends a JSON-encoded error response.
func (h *rawRPCHandler) sendError(w http.ResponseWriter, statusCode int, message string) error {
	force := true
	return rawrpc.SendErrorRawString(w, statusCode, message, force)
}
