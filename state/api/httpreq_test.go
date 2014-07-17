// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api_test

import (
	gc "launchpad.net/gocheck"

	simpleTesting "github.com/juju/juju/rpc/simple/testing"
	"github.com/juju/juju/state/api"
)

// TODO (ericsnow) BUG#1336542 Update the following to use the common code:
// charms_test.go
// tools_test.go
// debuglog_test.go

var _ = gc.Suite(&authHttpSuite{})

type authHttpSuite struct {
	clientSuite
	simpleTesting.HTTPReqSuite
}

func (s *authHttpSuite) SetUpSuite(c *gc.C) {
	s.clientSuite.SetUpSuite(c)
	s.HTTPReqSuite.SetUpSuite(c)
}

func (s *authHttpSuite) TearDownSuite(c *gc.C) {
	s.clientSuite.TearDownSuite(c)
	s.HTTPReqSuite.TearDownSuite(c)
}

func (s *authHttpSuite) SetUpTest(c *gc.C) {
	s.clientSuite.SetUpTest(c)
	s.HTTPReqSuite.SetUpTest(c)
}

func (s *authHttpSuite) TearDownTest(c *gc.C) {
	s.clientSuite.TearDownTest(c)
	s.HTTPReqSuite.TearDownTest(c)
}

func (s *authHttpSuite) APIClient() *api.Client {
	return s.clientSuite.APIState.Client()
}
