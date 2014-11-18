// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package http

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/state"
)

type newHandlerFunc func(HTTPHandler) http.Handler

// HTTPHandlers is the registry of HTTP handler factory functions.
var HTTPHandlers = make(map[string]newHandlerFunc)

// RegisterHTTPHandler adds an HTTP handler that will be handled by the
// API server.
func RegisterHTTPHandler(pattern string, newHandler newHandlerFunc) {
	HTTPHandlers[pattern] = newHandler
}

// HTTPHandlersLegacy is the registry of legacy HTTP handler factory functions.
var HTTPHandlersLegacy = make(map[string]newHandlerFunc)

// RegisterHTTPHandler adds an HTTP handler that will be handled by the
// API server.
func RegisterLegacyHTTPHandler(pattern string, newHandler newHandlerFunc) {
	HTTPHandlersLegacy[pattern] = newHandler
}

// HTTPHandler handles http requests through HTTPS in the API server.
type HTTPHandler struct {
	// State is the juju state used for the handled request.
	State *state.State
	// CheckAuth is used to authenticate HTTP requests.
	CheckAuth func(params.LoginRequest) error
	// DataDir is where juju data files are located.
	DataDir string
	// LogDir is where juju log files are located.
	LogDir string
}

// Authenticate parses HTTP basic authentication and authorizes the
// request by looking up the provided tag and password against state.
func (h *HTTPHandler) Authenticate(r *http.Request) error {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || parts[0] != "Basic" {
		// Invalid header format or no header provided.
		return fmt.Errorf("invalid request format")
	}
	// Challenge is a base64-encoded "tag:pass" string.
	// See RFC 2617, Section 2.
	challenge, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("invalid request format")
	}
	tagPass := strings.SplitN(string(challenge), ":", 2)
	if len(tagPass) != 2 {
		return fmt.Errorf("invalid request format")
	}
	// Only allow users, not agents.
	if _, err := names.ParseUserTag(tagPass[0]); err != nil {
		return errors.Errorf("invalid tag: %v", tagPass[0])
	}
	// Ensure the credentials are correct.
	err = h.CheckAuth(params.LoginRequest{
		AuthTag:     tagPass[0],
		Credentials: tagPass[1],
	})
	return err
}

func (h *HTTPHandler) getEnvironUUID(r *http.Request) string {
	return r.URL.Query().Get(":envuuid")
}

// ValidateEnvironUUID validates that the requested environment UUID
// matches the current environment.
func (h *HTTPHandler) ValidateEnvironUUID(r *http.Request) error {
	// Note: this is only true until we have support for multiple
	// environments. For now, there is only one, so we make sure that is
	// the one being addressed.
	envUUID := h.getEnvironUUID(r)
	logger.Tracef("got a request for env %q", envUUID)
	if envUUID == "" {
		return nil
	}
	env, err := h.State.Environment()
	if err != nil {
		logger.Infof("error looking up environment: %v", err)
		return err
	}
	if env.UUID() != envUUID {
		logger.Infof("environment uuid mismatch: %v != %v", envUUID, env.UUID())
		return errors.Errorf("unknown environment: %q", envUUID)
	}
	return nil
}

// errorSender implementations send errors back to the caller.
type errorSender interface {
	// SendError sends the error message as an HTTP response.
	SendError(w http.ResponseWriter, statusCode int, message string)
}

// AuthError sends an unauthorized error.
func (h *HTTPHandler) AuthError(w http.ResponseWriter, sender errorSender) {
	w.Header().Set("WWW-Authenticate", `Basic realm="juju"`)
	sender.SendError(w, http.StatusUnauthorized, "unauthorized")
}
