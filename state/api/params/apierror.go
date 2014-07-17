// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package params

import (
	"fmt"

	"github.com/juju/juju/rpc"
	"github.com/juju/juju/rpc/errors"
	"github.com/juju/juju/rpc/errors/errorcodes"
)

//---------------------------
// errors

// Error is the type of error returned by any call to the state API.
//
// We use a distinct error type to ensure that the error string remains
// minimal.  Also, it's best not to make clients know that we're using
// the rpc package.
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

// GoString implements fmt.GoStringer.  It means that a *Error shows its
// contents correctly when printed with %#v.
func (e Error) GoString() string {
	return fmt.Sprintf("&params.Error{%q, %q}", e.Code, e.Message)
}

// ClientError maps errors returned from an RPC call into local errors with
// appropriate values.
func ClientError(err error) error {
	rerr, ok := err.(*rpc.RequestError)
	if !ok {
		return err
	}
	// We use our own error type rather than rpc.ServerError
	// because we don't want the code or the "server error" prefix
	// within the error message. Also, it's best not to make clients
	// know that we're using the rpc package.
	return &Error{
		Message: rerr.Message,
		Code:    rerr.Code,
	}
}

//---------------------------
// error codes

// The Code constants hold error codes for some kinds of error.
const (
	CodeNotFound            = errorcodes.NotFound
	CodeUnauthorized        = errorcodes.Unauthorized
	CodeCannotEnterScope    = "cannot enter scope"
	CodeCannotEnterScopeYet = "cannot enter scope yet"
	CodeExcessiveContention = "excessive contention"
	CodeUnitHasSubordinates = "unit has subordinates"
	CodeNotAssigned         = "not assigned"
	CodeStopped             = "stopped"
	CodeHasAssignedUnits    = "machine has assigned units"
	CodeNotProvisioned      = "not provisioned"
	CodeNoAddressSet        = "no address set"
	CodeTryAgain            = errorcodes.TryAgain
	CodeNotImplemented      = errorcodes.NotImplemented
	CodeAlreadyExists       = "already exists"
)

var ErrCode = errors.ErrCode

var (
	IsCodeNotFound            = errors.ErrorCodeChecker(CodeNotFound)
	IsCodeUnauthorized        = errors.ErrorCodeChecker(CodeUnauthorized)
	IsCodeCannotEnterScope    = errors.ErrorCodeChecker(CodeCannotEnterScope)
	IsCodeCannotEnterScopeYet = errors.ErrorCodeChecker(CodeCannotEnterScopeYet)
	IsCodeExcessiveContention = errors.ErrorCodeChecker(CodeExcessiveContention)
	IsCodeUnitHasSubordinates = errors.ErrorCodeChecker(CodeUnitHasSubordinates)
	IsCodeNotAssigned         = errors.ErrorCodeChecker(CodeNotAssigned)
	IsCodeStopped             = errors.ErrorCodeChecker(CodeStopped)
	IsCodeHasAssignedUnits    = errors.ErrorCodeChecker(CodeHasAssignedUnits)
	IsCodeNotProvisioned      = errors.ErrorCodeChecker(CodeNotProvisioned)
	IsCodeNoAddressSet        = errors.ErrorCodeChecker(CodeNoAddressSet)
	IsCodeTryAgain            = errors.ErrorCodeChecker(CodeTryAgain)
	IsCodeNotImplemented      = errors.ErrorCodeChecker(CodeNotImplemented)
	IsCodeAlreadyExists       = errors.ErrorCodeChecker(CodeAlreadyExists)
)

// IsCodeNotFoundOrCodeUnauthorized is used in API clients which,
// pre-API, used errors.IsNotFound; this is because an API client is
// not necessarily privileged to know about the existence or otherwise
// of a particular entity, and the server may hence convert NotFound
// to Unauthorized at its discretion.
func IsCodeNotFoundOrCodeUnauthorized(err error) bool {
	return IsCodeNotFound(err) || IsCodeUnauthorized(err)
}
