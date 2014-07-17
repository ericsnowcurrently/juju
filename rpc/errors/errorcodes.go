// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package errors

// XXX Change error codes from strings to their own type (aliasing string).

// ErrorCoder represents an any error that has an associated
// error code. An error code is a short string that represents the
// kind of an error.
type ErrorCoder interface {
	ErrorCode() string
}

// ErrCode returns the error code associated with
// the given error, or the empty string if there
// is none.
func ErrCode(err error) string {
	if err, _ := err.(ErrorCoder); err != nil {
		return err.ErrorCode()
	}
	return ""
}

// ErrorCodeChecker returns a function that may be used to check errors
// for a particular error code.
func ErrorCodeChecker(code string) func(error) bool {
	return func(err error) bool {
		return ErrCode(err) == code
	}
}
