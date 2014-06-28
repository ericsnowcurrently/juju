// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package errorcodes

import (
	"github.com/juju/juju/rpc"
	"github.com/juju/juju/state/api"
)

// The Code constants hold error codes for some kinds of error.
const (
	CodeNotFound            api.ErrorCode = "not found"
	CodeUnauthorized        api.ErrorCode = "unauthorized access"
	CodeCannotEnterScope    api.ErrorCode = "cannot enter scope"
	CodeCannotEnterScopeYet api.ErrorCode = "cannot enter scope yet"
	CodeExcessiveContention api.ErrorCode = "excessive contention"
	CodeUnitHasSubordinates api.ErrorCode = "unit has subordinates"
	CodeNotAssigned         api.ErrorCode = "not assigned"
	CodeStopped             api.ErrorCode = "stopped"
	CodeHasAssignedUnits    api.ErrorCode = "machine has assigned units"
	CodeNotProvisioned      api.ErrorCode = "not provisioned"
	CodeNoAddressSet        api.ErrorCode = "no address set"
	CodeTryAgain            api.ErrorCode = "try again"
	CodeNotImplemented      api.ErrorCode = rpc.CodeNotImplemented
	CodeAlreadyExists       api.ErrorCode = "already exists"
)
