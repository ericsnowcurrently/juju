// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// XXX Move this into state/api (or a state/api/client package)?

package api

import (
	"github.com/juju/juju/rpc"
	ec "github.com/juju/juju/state/api/errorcodes"
)

/*
//---------------------------
// errors

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
*/

//---------------------------
// error codes

func IsCodeNoError(err error) bool {
	return rpc.ErrCode(err) == ec.CodeNoError
}

func IsCodeNotFound(err error) bool {
	return rpc.ErrCode(err) == ec.CodeNotFound
}

func IsCodeUnauthorized(err error) bool {
	return rpc.ErrCode(err) == ec.CodeUnauthorized
}

// IsCodeNotFoundOrCodeUnauthorized is used in API clients which,
// pre-API, used errors.IsNotFound; this is because an API client is
// not necessarily privileged to know about the existence or otherwise
// of a particular entity, and the server may hence convert NotFound
// to Unauthorized at its discretion.
func IsCodeNotFoundOrCodeUnauthorized(err error) bool {
	return IsCodeNotFound(err) || IsCodeUnauthorized(err)
}

func IsCodeCannotEnterScope(err error) bool {
	return rpc.ErrCode(err) == ec.CodeCannotEnterScope
}

func IsCodeCannotEnterScopeYet(err error) bool {
	return rpc.ErrCode(err) == ec.CodeCannotEnterScopeYet
}

func IsCodeExcessiveContention(err error) bool {
	return rpc.ErrCode(err) == ec.CodeExcessiveContention
}

func IsCodeUnitHasSubordinates(err error) bool {
	return rpc.ErrCode(err) == ec.CodeUnitHasSubordinates
}

func IsCodeNotAssigned(err error) bool {
	return rpc.ErrCode(err) == ec.CodeNotAssigned
}

func IsCodeStopped(err error) bool {
	return rpc.ErrCode(err) == ec.CodeStopped
}

func IsCodeHasAssignedUnits(err error) bool {
	return rpc.ErrCode(err) == ec.CodeHasAssignedUnits
}

func IsCodeNotProvisioned(err error) bool {
	return rpc.ErrCode(err) == ec.CodeNotProvisioned
}

func IsCodeNoAddressSet(err error) bool {
	return rpc.ErrCode(err) == ec.CodeNoAddressSet
}

func IsCodeTryAgain(err error) bool {
	return rpc.ErrCode(err) == ec.CodeTryAgain
}

func IsCodeNotImplemented(err error) bool {
	return rpc.ErrCode(err) == ec.CodeNotImplemented
}

func IsCodeAlreadyExists(err error) bool {
	return rpc.ErrCode(err) == ec.CodeAlreadyExists
}
