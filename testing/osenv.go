// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"os"
	"strings"

	"github.com/juju/testing"
	"github.com/juju/utils"
	"github.com/juju/utils/featureflag"
	"github.com/juju/utils/set"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/juju/osenv"
)

var envWhitelist = []string{
	//"PATH",

	//"GOPATH",
	//"JOB",
	//"LOGNAME",
	//"SHLVL",
	//"USER",

	//"TEXTDOMAIN",
	//"BASH_ENV",
	//"SHELL",
	//"TTY",
	//"TEXTDOMAINDIR",
	//"LANGUAGE",
	//"OLDPWD",
	//"PWD",
	//"HOME",
	//"BLOCKSIZE",
	//"LANG",
	//"TERM",

	//osenv.JujuEnvEnvKey,
	//osenv.JujuLoggingConfigEnvKey,
	//osenv.JujuHomeEnvKey,
	osenv.JujuRepositoryEnvKey,
	//"JUJU_ENV",
	//"JUJU_REPOSITORY",
	//"JUJU_HOME",
	"JUJU_CONTEXT_ID",
	"JUJU_AGENT_SOCKET",
	"JUJU_DUMMY_DELAY",

	"UPSTART_JOB",
	"PATH",
	"APPDATA",
	"USER",
	"SUDO_USER",
	"SSH_PASS",
	"PSModulePath",
	"LC_ALL",
}

//var envBlacklist = []string{
//	osenv.JujuHomeEnvKey,
//	osenv.JujuEnvEnvKey,
//	osenv.JujuLoggingConfigEnvKey,
//	osenv.JujuFeatureFlagEnvKey,
//}

type jujuOSEnvSuite struct {
	oldJujuHome string
}

func (s *jujuOSEnvSuite) SetUpSuite(c *gc.C) {
}

func (s *jujuOSEnvSuite) TearDownSuite(c *gc.C) {
}

func (s *jujuOSEnvSuite) SetUpTest(c *gc.C) {
	utils.SetHome("")
	s.oldJujuHome = osenv.SetJujuHome("")
	// Update the feature flag set to be empty (given we have just set the
	// environment value to the empty string)
	featureflag.SetFlagsFromEnvironment(osenv.JujuFeatureFlagEnvKey)
}

func (s *jujuOSEnvSuite) TearDownTest(c *gc.C) {
	osenv.SetJujuHome(s.oldJujuHome)
}

func (s *jujuOSEnvSuite) ExtendEnvWhitelist(whitelist set.Strings) {
	for _, envVar := range envWhitelist {
		whitelist.Add(envVar)
	}
}

func (s *jujuOSEnvSuite) SetFeatureFlags(flag ...string) {
	flags := strings.Join(flag, ",")
	if err := os.Setenv(osenv.JujuFeatureFlagEnvKey, flags); err != nil {
		panic(err)
	}
	logger.Debugf("setting feature flags: %s", flags)
	featureflag.SetFlagsFromEnvironment(osenv.JujuFeatureFlagEnvKey)
}

// JujuOSEnvSuite isolates the tests from Juju environment variables.
// This is intended to be only used by existing suites, usually embedded in
// BaseSuite and in FakeJujuHomeSuite. Eventually the tests relying on
// JujuOSEnvSuite will be converted to use the IsolationSuite in
// github.com/juju/testing, and this suite will be removed.
// Do not use JujuOSEnvSuite when writing new tests.
type JujuOSEnvSuite struct {
	testing.OsEnvSuite
	jujuOSEnvSuite
}

func (s *JujuOSEnvSuite) SetUpSuite(c *gc.C) {
	s.OsEnvSuite.SetUpSuite(c)
	s.jujuOSEnvSuite.SetUpSuite(c)

	s.ExtendEnvWhitelist(s.EnvWhitelist)
}

func (s *JujuOSEnvSuite) TearDownSuite(c *gc.C) {
	s.jujuOSEnvSuite.TearDownSuite(c)
	s.OsEnvSuite.TearDownSuite(c)
}

func (s *JujuOSEnvSuite) SetUpTest(c *gc.C) {
	s.OsEnvSuite.SetUpTest(c)
	s.jujuOSEnvSuite.SetUpTest(c)
}

func (s *JujuOSEnvSuite) TearDownTest(c *gc.C) {
	s.jujuOSEnvSuite.TearDownTest(c)
	s.OsEnvSuite.TearDownTest(c)
}
