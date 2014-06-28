// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package params

import (
	//	"fmt"

	"github.com/juju/juju/rpc"
)

// XXX Move the rest to state/api/errors.go?

// Error is the type of error returned by any call to the state API.
// We use our own error type rather than rpc.ServerError
// because we don't want the code or the "server error" prefix
// within the error message. Also, it's best not to make clients
// know that we're using the rpc package.
type Error rpc.Error

func (e *Error) Error() string {
	return e.Message
}

// ClientError maps errors returned from an RPC call into local errors with
// appropriate values.
func ClientError(err error) error {
	rerr, ok := err.(*rpc.RequestError)
	if !ok {
		return err
	}
	return &Error{
		Message: rerr.Message,
		Code:    rerr.Code,
	}
}

/*
// Error is the type of error returned by any call to the state API
type Error struct {
	Message string
	Code    string
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) ErrorCode() string {
	return e.Code
}

var _ rpc.ErrorCoder = (*Error)(nil)

// GoString implements fmt.GoStringer.  It means that a *Error shows its
// contents correctly when printed with %#v.
func (e Error) GoString() string {
	return fmt.Sprintf("&params.Error{%q, %q}", e.Code, e.Message)
}
*/
