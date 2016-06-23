// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rfc1123

const (
	hostnameMax = 255 // octets (strict value is 63)

	namePattern     = `[a-zA-Z0-9](?:[-a-zA-Z0-9]*[a-zA-Z0-9])?`
	hostnamePattern = namePattern + `(?:` + namePattern + `)*`
)

var hostnameRegex = regexp.MustCompile("^" + hostnamePattern + "$")

// See https://tools.ietf.org/html/rfc1123#section-2 (2.1).
type Hostname struct {
	name string
}

func NewHostname(name string) (Hostname, error) {
	h := Hostname{name}
	return h, h.validate()
}

func (h Hostname) DomainName() rfc1035.DomainName {
	// An RFC 952 hostname is necessarily compatible with RFC 1035.
	// ...except if it starts with a digit.
	dn, _ := rfc1035.ParseDomainName(h.name)
	return dn
}

func (h Hostname) validate() error {
	if !hostnameRegex.MatchString(h.name) {
		return fmt.Errorf("unsupported name")
	}
	if h.name == "" {
		return fmt.Errorf("empty hostname")
	}
	if len(h.name) > hostnameMax {
		return fmt.Errorf("hostname too big (%d octets > %d max)", len(h.name), hostnameMax)
	}
	return nil
}
