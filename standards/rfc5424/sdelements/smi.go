// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package sdelements

//func (pen PrivateEnterpriseNumber) Code() SMINetworkManagementCode {
//	return SMINetworkManagementCode{
//		Prefix: privateEnterprisePrefix,
//		Number: int(PrivateEnterpriseNumber),
//	}
//}

/*
// See http://www.iana.org/assignments/smi-numbers/smi-numbers.xhtml.
type SMINetworkManagementCode interface {
    SubTree(string, int) (SMINetworkManagementCode, error) {
}

type smiNetworkManagementCode struct {
	Prefix SMINetworkManagementCode
	Name   String
	Number int
}

var (
	zeroSMINetworkManagementCode SMINetworkManagementCode
	privateEnterprisePrefix      = SMINetworkManagementCode{
		Prefix: SMINetworkManagementCode{
			Prefix: SMINetworkManagementCode{
				Prefix: SMINetworkManagementCode{
					Prefix: SMINetworkManagementCode{
						Prefix: SMINetworkManagementCode{
							Name:   "iso",
							Number: 1,
						},
						Name:   "org",
						Number: 3,
					},
					Name:   "dod",
					Number: 6,
				},
				Name:   "internet",
				Number: 1,
			},
			Name:   "private",
			Number: 4,
		},
		Name:   "enterprise",
		Number: 1,
	}
)

func (smiCode SMINetworkManagementCode) resolve() []uint8 {
	if smiCode == zeroSMINetworkManagementCode {
		return nil
	}

	if smiCode.Number == 0 {
		return smiCode.Prefix.resolve()
	}
	return append(smiCode.Prefix.resolve(), smiCode.Number)
}

func (smiCode SMINetworkManagementCode) resolveName() []string {
	if smiCode == zeroSMINetworkManagementCode {
		return nil
	}

	name := smiCode.Name
	if name == "" {
		if smiCode.Number == 0 {
			return smiCode.Prefix.resolveName()
		}
		name = fmt.Sprint(smiCode.Number)
	}
	return append(smiCode.Prefix.resolveName(), name)
}

func (smiCode SMINetworkManagementCode) Namespace() string {
	return strings.Join(smiCode.resolveName(), ".")
}

func (smiCode SMINetworkManagementCode) String() string {
	return strings.Join(smiCode.resolve(), ".")
}

func (smiCode SMINetworkManagementCode) Validate() error {
	if smiCode == zeroSMINetworkManagementCode {
		return nil
	}

	// TODO(ericsnow) check the name

	if smiCode.Number <= 0 {
		return fmt.Errorf("negative Number")
	}

	return smiCode.Prefix.Validate()
}
*/
