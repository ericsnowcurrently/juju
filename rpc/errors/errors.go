// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package errors

type FailableResult interface {
	Err() error
}

// BaseError is a simple base for API-related errors.
type BaseError struct {
	Message string
	Code    string
}

func (e *BaseError) Error() string {
	return e.Message
}

func (e *BaseError) ErrorCode() string {
	return e.Code
}

func (e *BaseError) Err() error {
	return e
}

// Verify that BaseError implements ErrorCoder.
var _ ErrorCoder = (*BaseError)(nil)

// APIError is the type of error returned by any call to the API.
type APIError struct {
	BaseError
}

func NewAPIError(msg, code string) *APIError {
	err := APIError{BaseError{msg, code}}
	return &err
}

// Verify that APIError implements ErrorCoder.
var _ ErrorCoder = (*APIError)(nil)
