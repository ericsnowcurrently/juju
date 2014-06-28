// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package errorcodes

import (
	c "github.com/juju/juju/state/api/client"
)

const (
	PreprocessingError   c.ErrorCode = "error while preparing request"
	RequestNotCreated    c.ErrorCode = "could not create API request"
	RequestNotSent       c.ErrorCode = "could not send API request"
	MethodNotSupported   c.ErrorCode = "method not supported by API server"
	ErrorDataNotRead     c.ErrorCode = "could not read raw error data"
	ErrorDataNotUnpacked c.ErrorCode = "could not unpack raw error data"
	RequestFailed        c.ErrorCode = "request failed on remote server"
	PostProcessingError  c.ErrorCode = "error while handling result"
)
