// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client

import (
	"github.com/juju/juju/state/api"
	"github.com/juju/juju/state/api/errorcodes"
)

//---------------------------
// client error codes

type ErrorCode api.ErrorCode

func IsPreprocessingError(err error) bool {
	return params.ErrCode(err) == errorcodes.PreprocessingError
}

func IsRequestNotCreated(err error) bool {
	return params.ErrCode(err) == errorcodes.RequestNotCreated
}

func IsRequestNotSent(err error) bool {
	return params.ErrCode(err) == errorcodes.RequestNotSent
}

func IsMethodNotSupported(err error) bool {
	return params.ErrCode(err) == errorcodes.MethodNotSupported
}

func IsErrorDataNotRead(err error) bool {
	return params.ErrCode(err) == errorcodes.ErrorDataNotRead
}

func IsErrorDataNotUnpacked(err error) bool {
	return params.ErrCode(err) == errorcodes.ErrorDataNotUnpacked
}

func IsRequestFailed(err error) bool {
	return params.ErrCode(err) == errorcodes.RequestFailed
}

func IsPostProcessingError(err error) bool {
	return params.ErrCode(err) == errorcodes.PostProcessingError
}

//---------------------------
// client errors

// We use our own error type rather than rpc.ServerError (or api.Error)
// because we don't want the code or the "server error" prefix
// within the error message. Also, it's best not to make clients
// know that we're using the rpc package.

type Error api.Error

func (e *Error) Error() string {
	return e.Message
}

// ClientError maps errors returned from an RPC call into local errors with
// appropriate values.
func ClientError(err error) *Error {
	rerr, ok := err.(*api.Error)
	if ok {
		return &Error{
			Message: rerr.Message,
			Code:    rerr.Code,
		}
	} else {
		return &Error{
			Message: err.Error(),
			//Code:    rerr.Code,
			Cause: err,
		}
	}
}
