// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package server

// TODO(ericsnow) Eliminate the apiserver/common import if possible.

import (
	"github.com/juju/errors"
	"github.com/juju/loggo"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/process"
	"github.com/juju/juju/process/api"
)

var hclogger = loggo.GetLogger("juju.process.api.server.hookcontext")

// UnitProcesses exposes the State functionality for a unit's
// workload processes.
type UnitProcesses interface {
	// Register registers a workload process for the unit and info.
	Register(info process.Info) error
	// List returns information on the process with the id on the unit.
	List(ids ...string) ([]process.Info, error)
	// Settatus sets the status for the process with the given id on the unit.
	SetStatus(id string, status process.Status) error
	// Unregister removes the information for the process with the given id.
	Unregister(id string) error
}

// HookContextAPI serves workload process-specific API methods.
type HookContextAPI struct {
	// State exposes the workload process aspect of Juju's state.
	State UnitProcesses
}

// NewHookContextAPI builds a new facade for the given State.
func NewHookContextAPI(st UnitProcesses) *HookContextAPI {
	return &HookContextAPI{State: st}
}

// RegisterProcess registers a workload process in state.
func (a HookContextAPI) RegisterProcesses(args api.RegisterProcessesArgs) (api.ProcessResults, error) {
	r := api.ProcessResults{}
	for _, apiProc := range args.Processes {
		info := api.API2Proc(apiProc)
		res := api.ProcessResult{
			ID: info.ID(),
		}
		hclogger.Debugf("registering %q: %#v", info.ID(), info)
		if err := a.State.Register(info); err != nil {
			hclogger.Debugf("error registering %q: %v", info.ID(), err)
			res.Error = common.ServerError(errors.Trace(err))
			r.Error = common.ServerError(api.BulkFailure)
		}

		r.Results = append(r.Results, res)
	}
	return r, nil
}

// ListProcesses builds the list of workload processes registered for
// the given unit and IDs. If no IDs are provided then all registered
// processes for the unit are returned.
func (a HookContextAPI) ListProcesses(args api.ListProcessesArgs) (api.ListProcessesResults, error) {
	var r api.ListProcessesResults

	if len(args.IDs) > 0 {
		hclogger.Debugf("listing %v", args.IDs)
	} else {
		hclogger.Debugf("listing all")
	}

	ids := args.IDs
	procs, err := a.State.List(ids...)
	if err != nil {
		r.Error = common.ServerError(err)
		return r, nil
	}

	if len(ids) == 0 {
		for _, proc := range procs {
			ids = append(ids, proc.ID())
		}
	}

	for _, id := range ids {
		res := api.ListProcessResult{
			ID: id,
		}

		found := false
		for _, proc := range procs {
			procID := proc.Name
			if proc.Details.ID != "" {
				procID += "/" + proc.Details.ID
			}
			if id == proc.ID() {
				res.Info = api.Proc2api(proc)
				found = true
				break
			}
		}
		if !found {
			hclogger.Debugf("error: %q not found", id)
			res.Error = common.ServerError(errors.NotFoundf("process %q", id))
			r.Error = common.ServerError(api.BulkFailure)
		}
		r.Results = append(r.Results, res)
	}
	return r, nil
}

// SetProcessesStatus sets the raw status of a workload process.
func (a HookContextAPI) SetProcessesStatus(args api.SetProcessesStatusArgs) (api.ProcessResults, error) {
	r := api.ProcessResults{}
	for _, arg := range args.Args {
		res := api.ProcessResult{
			ID: arg.ID,
		}
		status := api.APIStatus2Status(arg.Status)
		hclogger.Debugf("setting status for %q to %#v", arg.ID, status)
		err := a.State.SetStatus(arg.ID, status)
		if err != nil {
			hclogger.Debugf("error setting status for %q: %v", arg.ID, err)
			res.Error = common.ServerError(err)
			r.Error = common.ServerError(api.BulkFailure)
		}
		r.Results = append(r.Results, res)
	}
	return r, nil
}

// UnregisterProcesses marks the identified process as unregistered.
func (a HookContextAPI) UnregisterProcesses(args api.UnregisterProcessesArgs) (api.ProcessResults, error) {
	hclogger.Debugf("unregistering %v", args.IDs)

	r := api.ProcessResults{}
	for _, id := range args.IDs {
		res := api.ProcessResult{
			ID: id,
		}
		if err := a.State.Unregister(id); err != nil {
			hclogger.Debugf("error unregistering %q: %v", id, err)
			res.Error = common.ServerError(errors.Trace(err))
			r.Error = common.ServerError(api.BulkFailure)
		}
		r.Results = append(r.Results, res)
	}
	return r, nil
}
