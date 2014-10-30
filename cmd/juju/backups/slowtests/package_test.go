// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package slowtests_test

import (
	"testing"

	jujutesting "github.com/juju/juju/testing"
)

func TestPackage(t *testing.T) {
	jujutesting.MgoTestPackage(t)
}
