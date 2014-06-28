// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package errorcodes

import (
	"github.com/juju/juju/rpc"
)

// The Code constants hold error codes for some kinds of error.
const (
	CodeNoError             rpc.ErrorCode = rpc.CodeNoError
	CodeNotFound            rpc.ErrorCode = "not found"
	CodeUnauthorized        rpc.ErrorCode = "unauthorized access"
	CodeCannotEnterScope    rpc.ErrorCode = "cannot enter scope"
	CodeCannotEnterScopeYet rpc.ErrorCode = "cannot enter scope yet"
	CodeExcessiveContention rpc.ErrorCode = "excessive contention"
	CodeUnitHasSubordinates rpc.ErrorCode = "unit has subordinates"
	CodeNotAssigned         rpc.ErrorCode = "not assigned"
	CodeStopped             rpc.ErrorCode = "stopped"
	CodeHasAssignedUnits    rpc.ErrorCode = "machine has assigned units"
	CodeNotProvisioned      rpc.ErrorCode = "not provisioned"
	CodeNoAddressSet        rpc.ErrorCode = "no address set"
	CodeTryAgain            rpc.ErrorCode = "try again"
	CodeNotImplemented      rpc.ErrorCode = rpc.CodeNotImplemented
	CodeAlreadyExists       rpc.ErrorCode = "already exists"
)
