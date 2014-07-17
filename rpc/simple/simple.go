// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package simple

import (
	"fmt"
	"net/http"

	"github.com/juju/juju/rpc/errors"
	"github.com/juju/juju/rpc/errors/errorcodes"
)

// TODO (ericsnow) BUG#1336542 Update the following to use the common code:
// charms_test.go
// tools_test.go
// debuglog_test.go

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func handleFailure(resp *http.Response) (err error) {
	// Handle API errors.
	switch resp.StatusCode {
	case http.StatusMethodNotAllowed, http.StatusNotFound:
		// Handle a "not implemented" response.  (The API server is too
		// old so the method is not supported.)
		err = errors.NewAPIError(
			fmt.Sprintf("API method not supported by server"),
			errorcodes.NotImplemented,
		)
	default:
		var failure error
		failure, err = ReadJSONResponse(resp, &errors.APIError{})
		if err == nil {
			err = failure
		}
	}
	return err
}

// SendRequest uses client to send an HTTP request and returns the response.
func SendRequest(client HTTPDoer, req *http.Request) (*http.Response, error) {
	var err error

	// Send the request.
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send API request: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		return resp, nil
	} else {
		defer resp.Body.Close()
		err = handleFailure(resp)
		return resp, err
	}
}
