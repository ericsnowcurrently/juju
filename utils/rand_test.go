// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package utils_test

import (
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/utils"
)

func (s *UtilsSuite) TestRandomHex(c *gc.C) {
	value, err := utils.RandomHex(8)
	c.Assert(err, gc.IsNil)
	c.Assert(len(value), gc.Equals, 16)
}

func (s *UtilsSuite) TestRandomHexChars(c *gc.C) {
	value, err := utils.RandomHex(8)
	c.Assert(err, gc.IsNil)
	c.Assert(value, gc.Matches, `[a-z0-9]*`)
}

func (s *UtilsSuite) TestRandomHexUnique(c *gc.C) {
	value1, err := utils.RandomHex(8)
	c.Assert(err, gc.IsNil)
	value2, err := utils.RandomHex(8)
	c.Assert(err, gc.IsNil)
	c.Assert(value2, gc.Not(gc.Equals), value1)
}
