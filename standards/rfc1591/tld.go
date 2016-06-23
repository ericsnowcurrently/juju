// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rfc1591

import (
	"fmt"
	"reflect"

	"github.com/juju/juju/standards/rfc1035"
	"github.com/juju/juju/standards/rfc5890"
)

const (
	tldTypeDefault TLDType = ""

	TLDTypeGeneric           = "generic"
	TLDTypeGenericRestricted = "generic-restricted"
	TLDTypeCountryCode       = "country-code"
	TLDTypeSponsored         = "sponsored"
	TLDTypeInfrastructure    = "infrastructure"
	TLDTypeTest              = "test"
)

type TLDType string

func (typ TLDType) String() string {
	return string(typ)
}

func (typ TLDType) Validate() error {
	if typ == tldTypeDefault {
		return fmt.Errorf("zero value not supported")
	}
	return nil
}

// TLD describes a top-level internet domain, as registered with IANA.
type TLD struct {
	// Name is the domain name label for the TLD.
	Name rfc1035.Label

	// Type is the kind of TLD.
	Type TLDType

	// Org is the name/description of the supporting organization.
	Org string
}

func (tld TLD) IsRoot() bool {
	var rootTLD TLD
	return reflect.DeepEqual(tld, rootTLD)
}

func (tld TLD) Domain() rfc5890.DomainName {
	return rfc5890.NewDomainName(tld.Name)
}

func (tld TLD) Validate() error {
	if tld.IsRoot() {
		return nil
	}

	if err := rfc5890.ValidateLabel(tld.Name); err != nil {
		return fmt.Errorf("bad Name: %v", err)
	}
	if err := tld.Type.Validate(); err != nil {
		return fmt.Errorf("bad Type: %v", err)
	}
	if tld.Org == "" {
		return fmt.Errorf("empty Org")
	}
	return nil
}
