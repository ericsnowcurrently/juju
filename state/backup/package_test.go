// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// XXX Change to backup_test
package backup

import (
	"testing"

	gitjujutesting "github.com/juju/testing"
)

func Test(t *testing.T) {
	gitjujutesting.MgoTestPackage(t, nil)
}
