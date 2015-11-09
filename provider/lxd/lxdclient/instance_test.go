// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxdclient_test

import (
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/provider/lxd/lxdclient"
)

var _ = gc.Suite(&instanceSpecSuite{})

type instanceSpecSuite struct {
	lxdclient.BaseSuite
}

func (s *instanceSpecSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
}

func (s *instanceSpecSuite) TestValidateOkay(c *gc.C) {
	spec := lxdclient.InstanceSpec{
		Name:      "inst-1",
		ImageName: "ubuntu",
	}
	err := spec.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *instanceSpecSuite) TestValidateFull(c *gc.C) {
	spec := lxdclient.InstanceSpec{
		Name:      "inst-1",
		ImageName: "ubuntu",
		Profiles:  []string{"profile-1"},
		Ephemeral: true,
		Metadata:  map[string]string{"x": "y"},
	}
	err := spec.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *instanceSpecSuite) TestValidateZeroValue(c *gc.C) {
	var spec lxdclient.InstanceSpec
	err := spec.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *instanceSpecSuite) TestValidateMissingName(c *gc.C) {
	spec := lxdclient.InstanceSpec{
		ImageName: "ubuntu",
	}
	err := spec.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *instanceSpecSuite) TestValidateMissingImage(c *gc.C) {
	spec := lxdclient.InstanceSpec{
		Name: "inst-1",
	}
	err := spec.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}
