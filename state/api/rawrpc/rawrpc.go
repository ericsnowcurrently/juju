// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// These are state API client methods that use their own raw connections
// to the state machine rather than the state's RPC client.

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/juju/juju/rpc"
	"github.com/juju/juju/state/api/params"
)

// errors

const (
	CodeRequestNotCreated     rpc.ErrorCode = "could not create request"
	CodeRequestFailed         rpc.ErrorCode = "could not send request"
	CodeResponseNotRead       rpc.ErrorCode = "could not read response"
	CodeResponseNotUnpacked   rpc.ErrorCode = "could not unpack response"
	CodeRequestFailedRemotely rpc.ErrorCode = "failure on the API server"
)

type Error struct {
	*params.Error
	Cause error
}

func NewError(code rpc.ErrorCode, cause error) *Error {
	return &Error{
		&params.Error{Message: "", Code: code},
		cause,
	}
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	} else if e.Code != "" {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %v", e.Code, e.Cause)
		} else {
			return string(e.Code)
		}
	} else if e.Cause != nil {
		return e.Cause.Error()
	} else {
		return ""
	}
}

// args

type Args interface {
	AsURLArgs() url.Values
}

// results

type RemoteResult interface {
	RemoteError() error
}

// requests

type RequestData struct {
	Stream   io.Reader
	Mimetype string
}

type Request http.Request
type Client http.Client
type Response http.Response

func GetRequest(serverroot, method string, args Args, data *RequestData) (*Request, *Error) {
	url := fmt.Sprintf("%s/%s", serverroot, method)
	if args != nil {
		url += "?" + args.AsURLArgs().Encode()
	}

	req, err := http.NewRequest("POST", url, data.Stream)
	if err != nil {
		return nil, NewError(CodeRequestNotCreated, err)
	}

	if data.Mimetype != "" {
		req.Header.Set("Content-Type", data.Mimetype)
	}

	return req, nil
}

func SendRequest(client *Client, req *Request) (*Response, *Error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, NewError(CodeRequestFailed, err)
	}
	return resp, nil
}

func HandleResponse(resp *Response, result RemoteResult) *Error {
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		// API server is too old, so the RPC method is not supported;
		// notify the client.
		return Error{
			Message: "API method not supported by the server",
			Code:    params.CodeNotImplemented,
		}
	}

	// Parse the result.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return NewError(CodeResponseNotRead, err)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return NewError(CodeResponseNotUnpacked, err)
	}

	err = result.RemoteError()
	if err != nil {
		return NewError(CodeRequestFailedRemotely, result.RemoteError())
	}

	return nil
}
