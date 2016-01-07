// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common

import (
	"github.com/juju/testing"
)

var (
	MachineJobFromParams    = machineJobFromParams
	ValidateNewFacade       = validateNewFacade
	WrapNewFacade           = wrapNewFacade
	NilFacadeRecord         = facadeRecord{}
	EnvtoolsFindTools       = &envtoolsFindTools
	SendMetrics             = &sendMetrics
	MockableDestroyMachines = destroyMachines
)

type Patcher interface {
	PatchValue(dest, value interface{})
}

// SanitizeFacades patches Facades so that for the lifetime of the test we get
// a clean slate to work from, and will not accidentally overrite/mutate the
// real facade registry.
func SanitizeFacades(patcher Patcher) {
	emptyFacades := &FacadeRegistry{}
	patcher.PatchValue(&Facades, emptyFacades)
}

type Versions versions

func DescriptionFromVersions(name string, vers Versions) FacadeDescription {
	return descriptionFromVersions(name, versions(vers))
}

func SanitizeHTTPEndpointsRegistry(patch Patcher) {
	emptyEndpoints := NewHTTPEndpoints()
	testing.PatchValue(&httpEndpoints.endpoints, emptyEndpoints)
}

func ExposeHTTPEndpointsRegistry() []HTTPEndpointSpec {
	return httpEndpoints.endpoints.specs()
}
