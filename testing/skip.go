// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"fmt"
	"runtime"

	gc "gopkg.in/check.v1"

	"github.com/juju/juju/juju/arch"
)

// SkipIfPPC64EL skips the test if the arch is PPC64EL and the
// compiler is gccgo.
func SkipIfPPC64EL(c *gc.C, bugID string) {
	if runtime.Compiler == "gccgo" &&
		arch.NormaliseArch(runtime.GOARCH) == arch.PPC64EL {
		c.Skip(fmt.Sprintf("Test disabled on PPC64EL until fixed - see bug %s", bugID))
	}
}

// SkipIfI386 skips the test if the arch is I386.
func SkipIfI386(c *gc.C, bugID string) {
	if arch.NormaliseArch(runtime.GOARCH) == arch.I386 {
		c.Skip(fmt.Sprintf("Test disabled on I386 until fixed - see bug %s", bugID))
	}
}
