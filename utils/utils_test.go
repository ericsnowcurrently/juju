// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package utils_test

import (
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/testing"
)

var _ = gc.Suite(&UtilsSuite{})

type UtilsSuite struct {
	testing.FilesSuite
}
