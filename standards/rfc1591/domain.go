// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rfc1591

type DomainName interface {
	TLDer
	IsRoot() bool
}

// IsTLDValid determines whether or not the domain name has a valid
// registered TLD, per RFC 1591 and the IANA database.
func IsTLDValid(dn DomainName) bool {
	if dn.IsRoot() {
		return true
	}
	_, ok := DB.LookUpDomain(dn)
	return ok
}
