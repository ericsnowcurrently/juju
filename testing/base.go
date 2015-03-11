// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"strings"

	"github.com/juju/loggo"
	"github.com/juju/testing"
	"github.com/juju/utils"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/network"
	"github.com/juju/juju/wrench"
)

var logger = loggo.GetLogger("juju.testing")

// BaseSuite provides required functionality for all test suites
// when embedded in a gocheck suite type:
// - logger redirect
// - no outgoing network access
// - protection of user's home directory
// - scrubbing of env vars
type BaseSuite struct {
	testing.IsolationSuite
	jujuOSEnvSuite
}

func (s *BaseSuite) SetUpSuite(c *gc.C) {
	wrench.SetEnabled(false)
	s.IsolationSuite.SetUpSuite(c)
	s.jujuOSEnvSuite.SetUpSuite(c)

	s.jujuOSEnvSuite.ExtendEnvWhitelist(s.EnvWhitelist)
	s.PatchValue(&utils.OutgoingAccessAllowed, false)
}

func (s *BaseSuite) TearDownSuite(c *gc.C) {
	s.jujuOSEnvSuite.TearDownSuite(c)
	s.IsolationSuite.TearDownSuite(c)
}

func (s *BaseSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)
	s.jujuOSEnvSuite.SetUpTest(c)

	// We do this to isolate invocations of bash from pulling in the
	// ambient user environment, and potentially affecting the tests.
	// We can't always just use IsolationSuite because we still need
	// PATH and possibly a couple other envars.
	s.PatchEnvironment("BASH_ENV", "")
	network.ResetGobalPreferIPv6()
}

func (s *BaseSuite) TearDownTest(c *gc.C) {
	s.jujuOSEnvSuite.TearDownTest(c)
	s.IsolationSuite.TearDownTest(c)
}

// CheckString compares two strings. If they do not match then the spot
// where they do not match is logged.
func CheckString(c *gc.C, value, expected string) {
	if !c.Check(value, gc.Equals, expected) {
		diffStrings(c, value, expected)
	}
}

func diffStrings(c *gc.C, value, expected string) {
	// If only Go had a diff library.
	vlines := strings.Split(value, "\n")
	elines := strings.Split(expected, "\n")
	vsize := len(vlines)
	esize := len(elines)

	if vsize < 2 || esize < 2 {
		return
	}

	smaller := elines
	if vsize < esize {
		smaller = vlines
	}

	for i := range smaller {
		vline := vlines[i]
		eline := elines[i]
		if vline != eline {
			c.Logf("first mismatched line (%d/%d):", i, len(smaller))
			c.Log("expected: " + eline)
			c.Log("got:      " + vline)
			break
		}
	}
}
