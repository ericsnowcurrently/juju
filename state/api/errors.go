// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"fmt"
)

//---------------------------
// API error codes

// An error code is a short string that represents the kind of an error.
type ErrorCode string

func (ec *ErrorCode) String() string {
	return string(ec)
}

// Err returns a new Error based on the error code.
func (ec *ErrorCode) Err(cause error, msg string) Error {
	return Error{
		Message: msg,
		Code:    ec,
		Cause:   cause,
	}
}

const NoError *ErrorCode = nil

// ErrorCoder represents any error that has an associated error code.
// XXX "inherit" from error?
type ErrorCoder interface {
	ErrorCode() *ErrorCode
}

// ErrCode returns the error code associated with
// the given error, or NoError if there is none.
func ErrCode(err error) string {
	err, ok := err.(ErrorCoder)
	if ok {
		return err.ErrorCode()
	} else {
		return NoError
	}
}

func IsCodeNotFound(err error) bool {
	return ErrCode(err) == CodeNotFound
}

func IsCodeUnauthorized(err error) bool {
	return ErrCode(err) == CodeUnauthorized
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
	return ErrCode(err) == CodeCannotEnterScope
}

func IsCodeCannotEnterScopeYet(err error) bool {
	return ErrCode(err) == CodeCannotEnterScopeYet
}

func IsCodeExcessiveContention(err error) bool {
	return ErrCode(err) == CodeExcessiveContention
}

func IsCodeUnitHasSubordinates(err error) bool {
	return ErrCode(err) == CodeUnitHasSubordinates
}

func IsCodeNotAssigned(err error) bool {
	return ErrCode(err) == CodeNotAssigned
}

func IsCodeStopped(err error) bool {
	return ErrCode(err) == CodeStopped
}

func IsCodeHasAssignedUnits(err error) bool {
	return ErrCode(err) == CodeHasAssignedUnits
}

func IsCodeNotProvisioned(err error) bool {
	return ErrCode(err) == CodeNotProvisioned
}

//---------------------------
// API errors

// Error is the type of error returned by any call to the state API
type Error struct {
	Message string
	Code    *ErrorCode
	Cause   error
}

func (e *Error) Error() string {
	if e.Message != "" {
		if e.Code != NoError {
			return fmt.Sprintf("%s (%s)", e.Message, e.Code)
		} else {
			return e.Message
		}
	} else if e.Code != NoError {
		if e.Cause != nil {
			return fmt.Sprintf("(%s) %v", e.Code, e.Cause)
		} else {
			return e.Code.String()
		}
	} else if e.Cause != nil {
		return e.Cause.Error()
	} else {
		return ""
	}
}

// ErrorCode returns the error's code, satisfying ErrorCoder.
func (e *Error) ErrorCode() string {
	return e.Code
}

// GoString implements fmt.GoStringer.  It means that a *Error shows its
// contents correctly when printed with %#v.
func (e *Error) GoString() string {
	return fmt.Sprintf("&api.Error{%q, %q}", e.Code, e.Message)
}

// XXX What is this for?
var _ ErrorCoder = (*Error)(nil)
