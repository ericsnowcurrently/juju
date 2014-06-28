// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// XXX Move this to a distinct "api" package from which rpc and state/api
// can both import?

package rpc

import (
	"fmt"
)

//---------------------------
// error codes

type ErrorCode string

func (ec *ErrorCode) String() string {
	return string(*ec)
}

// XXX Move to own namespace (package).
const (
	CodeNoError        ErrorCode = ""
	CodeNotImplemented ErrorCode = "not implemented"
)

// ErrorCoder represents an any error that has an associated
// error code. An error code is a short string that represents the
// kind of an error.
type ErrorCoder interface {
	ErrorCode() ErrorCode
}

// ErrCode returns the error code associated with
// the given error, or CodeNoError if there
// is none.
func ErrCode(err error) ErrorCode {
	if err, _ := err.(ErrorCoder); err != nil {
		return err.ErrorCode()
	}
	return CodeNoError
}

//---------------------------
// errors

// Error is the type of error returned by any call to the state API
type Error struct {
	Message string
	Code    ErrorCode
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) ErrorCode() ErrorCode {
	return e.Code
}

// GoString implements fmt.GoStringer.  It means that a *Error shows its
// contents correctly when printed with %#v.
func (e Error) GoString() string {
	return fmt.Sprintf("&api.Error{%q, %q}", e.Code, e.Message)
}

// XXX What is this for?  Verify that Error implements ErrorCoder?
var _ ErrorCoder = (*Error)(nil)
