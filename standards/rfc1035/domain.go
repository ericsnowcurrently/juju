// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rfc1035

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	domainNameMax = 255 // octets
	labelMax      = 63  // octets

	LabelPattern = `[a-zA-Z](?:[-a-zA-Z0-9]*[a-zA-Z0-9])?`
)

var (
	// Root is the root domain for all domain names.
	Root DomainName
)

// DomainName identifies an internet domain. It
// A domain name is the root of a domain namespace (a tree).
type DomainName struct {
	domainName
}

func NewDomainName(path ...Label) DomainName {
	return DomainName{newDomainName(path)}
}

func ParseDomainName(val string) (DomainName, error) {
	names := strings.Split(val, ".")
	return Root.SubStrings(names...)
}

func (dn DomainName) FQDN() string {
	if dn.IsRoot() {
		return ""
	}
	return dn.String() + "."
}

func (dn DomainName) Octets() []byte {
	octets := dn.Path().Octets()
	return append(octets, 0)
}

func (dn DomainName) IsRoot() bool {
	return (len(dn.Path()) == 0)
}

func (dn DomainName) Domain() DomainName {
	return NewDomainName([]Label(dn.Path().Domain())...)
}

func (dn DomainName) TLD() Label {
	return dn.Path().TLD()
}

// Empty labels are ignored.
func (dn DomainName) Sub(subPath ...Label) (DomainName, bool) {
	sub, tooBig := dn.Path().Sub(subPath)
	return NewDomainName([]Label(sub)...), tooBig
}

func (dn DomainName) SubStrings(subPath ...string) (DomainName, error) {
	sub, _ := dn.Path().SubStrings(subPath) // We'll pick up the error in validate().
	return NewDomainName([]Label(sub)...), sub.Validate(checkASCIIName)
}

// ValidateDomainName ensures that the domain name satisfies RFC 1035.
//
// Note that this function is only necessary when an error from
// NewDomainName() or ParseDomainName() was ignored, or if invalid
// labels were passed to Sub().
func ValidateDomainName(dn DomainName) error {
	return dn.Path().Validate(checkASCIIName)
}

type RelativeDomainName struct {
	domainName
}

func NewRelativeDomainName(path ...Label) RelativeDomainName {
	return RelativeDomainName{newDomainName(path)}
}

func ParseRelativeDomainName(val string) (RelativeDomainName, error) {
	names := strings.Split(val, ".")
	return RelativeDomainName{}.SubStrings(names...)
}

func (dn RelativeDomainName) Qualify(domain DomainName) (DomainName, bool) {
	return domain.Sub(dn.path...)
}

func (dn RelativeDomainName) Octets() []byte {
	return dn.Path().Octets()
}

func (dn RelativeDomainName) Domain() RelativeDomainName {
	return NewRelativeDomainName([]Label(dn.Path().Domain())...)
}

func (dn RelativeDomainName) Sub(subPath ...Label) (RelativeDomainName, bool) {
	sub, tooBig := dn.Path().Sub(subPath)
	return NewRelativeDomainName([]Label(sub)...), tooBig
}

func (dn RelativeDomainName) SubStrings(subPath ...string) (RelativeDomainName, error) {
	sub, _ := dn.Path().SubStrings(subPath) // We'll pick up the error in validate().
	return NewRelativeDomainName([]Label(sub)...), sub.Validate(checkASCIIName)
}

// ValidateRelativeDomainName ensures that the domain name satisfies
// RFC 1035.
//
// Note that this function is only necessary when an error from
// NewRelativeDomainName() or ParseRelativeDomainName() was ignored,
// or if invalid labels were passed to Sub().
func ValidateRelativeDomainName(dn RelativeDomainName) error {
	return dn.Path().Validate(checkASCIIName)
}

type domainName struct {
	path []Label
}

func newDomainName(path []Label) domainName {
	dn := domainName{
		path: make([]Label, len(path)),
	}
	copy(dn.path, path)
	return dn
}

func (dn domainName) Path() DomainNamePath {
	path := make([]Label, len(dn.path))
	copy(path, dn.path)
	return DomainNamePath(path)
}

func (dn domainName) String() string {
	return dn.Path().String()
}

func (dn domainName) Name() Label {
	return dn.Path().Name()
}

type DomainNamePath []Label

func (path DomainNamePath) String() string {
	if len(path) == 0 {
		return ""
	}
	str := path[0].String()
	for _, name := range path[1:] {
		str += "." + name.String()
	}
	return str
}

func (path DomainNamePath) Octets() []byte {
	var octets []byte
	for _, name := range path {
		octets = append(octets, name.Octets()...)
	}
	return octets
}

func (path DomainNamePath) Name() Label {
	if len(path) == 0 {
		return Label{}
	}
	return path[0]
}

func (path DomainNamePath) Domain() DomainNamePath {
	if len(path) == 0 {
		return path
	}
	return path[1:]
}

func (path DomainNamePath) TLD() Label {
	if len(path) == 0 {
		return Label{}
	}
	return path[len(path)-1]
}

// We operate under the assumption that the provided labels are valid.
func (path DomainNamePath) Sub(subPath []Label) (DomainNamePath, bool) {
	// We will use *at least* what's in path.
	sub := make(DomainNamePath, len(path))
	for _, name := range subPath {
		if name.IsRoot() {
			// We ignore root labels here. They have no meaning except
			// as the root of a path, which is irrelevant at this point.
			continue
		}
		sub = append(sub, name)
	}
	sub = append(sub, path...)

	tooBig := (len(sub.Octets()) > domainNameMax-1) // 1 for the root
	return sub, tooBig
}

func (path DomainNamePath) SubStrings(subPath []string) (DomainNamePath, bool) {
	labels := make([]Label, len(subPath))
	for i, name := range subPath {
		labels[i] = Label{[]byte(name)}
	}
	return path.Sub(labels)
}

func (path DomainNamePath) Validate(checkName func(Label) error) error {
	for _, name := range path {
		if checkName != nil {
			if err := checkName(name); err != nil {
				return err
			}
		}
		if err := name.validate(); err != nil {
			return err
		}
	}

	size := len(path.Octets())
	if size > domainNameMax-1 { // 1 for the root
		return &errDomainNameTooBig{path.String(), size}
	}

	return nil
}

type errDomainNameTooBig struct {
	dn   string
	size int
}

func (err errDomainNameTooBig) Error() string {
	return fmt.Sprintf("domain name %q too big (%d octets > max %d)", err.dn, err.size, domainNameMax)
}

func IsDomainNameTooBig(err error) bool {
	switch err.(type) {
	case *errDomainNameTooBig, errDomainNameTooBig:
		return true
	default:
		return false
	}
}

// Label is a single name element in a domain name.
type Label struct {
	name []byte
}

func NewLabel(name string) (Label, error) {
	return NewLabelBytes([]byte(name), checkASCIIName)
}

func NewLabelBytes(name []byte, checkName func(Label) error) (Label, error) {
	label := Label{
		name: name,
	}
	if checkName != nil {
		if err := checkName(label); err != nil {
			return label, err
		}
	}
	return label, label.validate()
}

// String returns the RFC 1035 representation of the label.
func (label Label) String() string {
	return string(label.name)
}

// See https://tools.ietf.org/html/rfc1035#section-3.1.
func (label Label) Octets() []byte {
	lengthOctet := byte(len(label.name))
	return append([]byte{lengthOctet}, label.name...)
}

func (label Label) IsRoot() bool {
	return (len(label.name) == 0)
}

// See https://tools.ietf.org/html/rfc1035#section-2.3.1
// and https://tools.ietf.org/html/rfc1035#section-2.3.4.
func (label Label) validate() error {
	if len(label.name) > labelMax {
		return &errLabelTooBig{string(label.name), len(label.name)}
	}
	return nil
}

type errLabelTooBig struct {
	label string
	size  int
}

func (err errLabelTooBig) Error() string {
	return fmt.Sprintf("label %q too big (%d octets > max %d)", err.label, err.size, labelMax)
}

func IsLabelTooBig(err error) bool {
	switch err.(type) {
	case *errLabelTooBig, errLabelTooBig:
		return true
	default:
		return false
	}
}

// ValidateLabel ensures that the label satisfies RFC 1035. Note that
// this function is only necessary when an error from NewLabel() was
// ignored.
func ValidateLabel(label Label) error {
	if err := checkASCIIName(label); err != nil {
		return err
	}
	return ValidateLabelBytes(label)
}

var labelRegex = regexp.MustCompile("^" + LabelPattern + "$")

func checkASCIIName(label Label) error {
	name := label.String()
	if !labelRegex.MatchString(name) {
		return fmt.Errorf("unsupported label name %q", name)
	}
	return nil
}

func ValidateLabelBytes(label Label) error {
	return label.validate()
}
