// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package sdelements

import (
	"fmt"

	"github.com/juju/juju/standards/rfc5424"
)

// TimeQuality is an IANA-registered structured data element that gives
// extra information about the log message's time stamp.
//
// See https://tools.ietf.org/html/rfc5424#section-7.1.
type TimeQuality struct {
	TZKnown      bool
	IsSynced     bool
	SyncAccuracy int
}

// ID returns the SD-ID for this element.
func (sde TimeQuality) ID() rfc5424.StructuredDataName {
	return "timeQuality"
}

// Params returns the []SD-PARAM for this element.
func (sde TimeQuality) Params() []rfc5424.StructuredDataParam {
	tzKnown := "0"
	if sde.TZKnown {
		tzKnown = "1"
	}
	isSynced := "0"
	if sde.IsSynced {
		isSynced = "1"
	}

	params := []rfc5424.StructuredDataParam{{
		Name:  "tzKnown",
		Value: rfc5424.StructuredDataParamValue(tzKnown),
	}, {
		Name:  "isSynced",
		Value: rfc5424.StructuredDataParamValue(isSynced),
	}}

	if sde.IsSynced && sde.SyncAccuracy > 0 {
		params = append(params, rfc5424.StructuredDataParam{
			Name:  "syncAccuracy",
			Value: rfc5424.StructuredDataParamValue(fmt.Sprint(sde.SyncAccuracy)),
		})
	}

	return params
}

// Validate ensures that the element is correct.
func (sde TimeQuality) Validate() error {
	if sde.SyncAccuracy < 0 {
		fmt.Errorf("SyncAccuracy must be positive integer")
	}
	return nil
}
