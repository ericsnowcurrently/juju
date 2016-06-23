// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rfc5890

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/juju/juju/standards/rfc1035"
)

const (
	// See https://tools.ietf.org/html/rfc5890#section-2.3.2.1.
	unicodePattern = `.*` // TODO(ericsnow) finish
	LabelPattern   = "" +
		"(" + rfc1035.LabelPattern + ")" +
		"|" +
		"(" + unicodePattern + ")"
)

var (
	// Root is the root domain for all domain names.
	Root DomainName
)

type DomainName struct {
	rfc1035.DomainName
}

func NewDomainName(path ...rfc1035.Label) DomainName {
	return DomainName{rfc1035.NewDomainName(path...)}
}

func ParseDomainName(val string) (DomainName, error) {
	names := strings.Split(val, ".")
	return Root.SubStrings(names...)
}

func (dn DomainName) SubStrings(subPath ...string) (DomainName, error) {
	sub, _ := dn.Path().SubStrings(subPath)
	base := rfc1035.NewDomainName([]rfc1035.Label(sub)...)
	return DomainName{base}, sub.Validate(checkName)
}

func ValidateDomainName(dn DomainName) error {
	return dn.Path().Validate(checkName)
}

type RelativeDomainName struct {
	rfc1035.RelativeDomainName
}

func ParseRelativeDomainName(val string) (RelativeDomainName, error) {
	names := strings.Split(val, ".")
	return RelativeDomainName{}.SubStrings(names...)
}

func (dn RelativeDomainName) SubStrings(subPath ...string) (RelativeDomainName, error) {
	sub, _ := dn.Path().SubStrings(subPath)
	base := rfc1035.NewRelativeDomainName([]rfc1035.Label(sub)...)
	return RelativeDomainName{base}, sub.Validate(checkName)
}

func ValidateRelativeDomainName(dn RelativeDomainName) error {
	return dn.Path().Validate(checkName)
}

func NewLabel(name string) (rfc1035.Label, error) {
	return rfc1035.NewLabelBytes([]byte(name), checkName)
}

// ValidateLabel ensures that the label satisfies RFC 5890.
func ValidateLabel(label rfc1035.Label) error {
	return checkName(label)
}

var labelRegex = regexp.MustCompile("^" + LabelPattern + "$")

func checkName(label rfc1035.Label) error {
	name := label.String()
	if !labelRegex.MatchString(name) {
		return fmt.Errorf("unsupported label name %q", name)
	}
	return nil
}
