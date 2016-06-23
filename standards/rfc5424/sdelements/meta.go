// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package sdelements

import (
	"fmt"

	"github.com/juju/juju/standards/rfc5424"
)

// Meta is an IANA-registered structured data element that associates
// additional information with a log message.
//
// See https://tools.ietf.org/html/rfc5424#section-7.3.
type Meta struct {
	SequenceID SequenceID // positive int32
	SysUpTime  SysUpTime  // RFC 3418
	Language   Language   // RFC 4646
}

// ID returns the SD-ID for this element.
func (sde Meta) ID() rfc5424.StructuredDataName {
	return "meta"
}

// Params returns the []SD-PARAM for this element.
func (sde Meta) Params() []rfc5424.StructuredDataParam {
	var params []rfc5424.StructuredDataParam

	sequenceID := sde.SequenceID.String()
	if sequenceID != "" {
		params = append(params, rfc5424.StructuredDataParam{
			Name:  "sequenceID",
			Value: rfc5424.StructuredDataParamValue(sequenceID),
		})
	}

	sysUpTime := sde.SysUpTime.String()
	if sysUpTime != "" {
		params = append(params, rfc5424.StructuredDataParam{
			Name:  "sysUpTime",
			Value: rfc5424.StructuredDataParamValue(sysUpTime),
		})
	}

	language := sde.Language.String()
	if language != "" {
		params = append(params, rfc5424.StructuredDataParam{
			Name:  "language",
			Value: rfc5424.StructuredDataParamValue(language),
		})
	}

	return params
}

// Validate ensures that the element is correct.
func (sde Meta) Validate() error {
	// TODO(ericsnow) finish
	return nil
}

type SequenceID uint32

func (id SequenceID) String() string {
	if id == 0 {
		return ""
	}
	return fmt.Sprint(id)
}

type SysUpTime uint32

func (sut SysUpTime) String() string {
	if sut == 0 {
		return ""
	}
	return fmt.Sprint(sut)
}

type Language string

func (lang Language) String() string {
	return string(lang)
}
