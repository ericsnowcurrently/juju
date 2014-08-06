// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	stdtesting "testing"

	"github.com/juju/testing/mgo"
	gc "launchpad.net/gocheck"
)

var MgoServer *mgo.MgoServer

// MgoTestPackage should be called to register the tests for any package
// that requires a secure connection to a MongoDB server.
func MgoTestPackage(t *stdtesting.T) {
	// TODO(ericsnow) don't hard-code the path.
	mgo.MgoTestPackage(t, Certs, "/usr/lib/juju/bin/mongod")
}

type MgoSuite struct {
	mgo.MgoSuite
}

func (s *MgoSuite) SetUpSuite(c *gc.C) {
	s.MgoSuite.SetUpSuite(c)
	MgoServer = s.Server()
}

type BaseMgoSuite struct {
	BaseSuite
	MgoSuite
}

func (s *BaseMgoSuite) SetUpSuite(c *gc.C) {
	s.BaseSuite.SetUpSuite(c)
	s.MgoSuite.SetUpSuite(c)
}

func (s *BaseMgoSuite) TearDownSuite(c *gc.C) {
	s.MgoSuite.TearDownSuite(c)
	s.BaseSuite.TearDownSuite(c)
}

func (s *BaseMgoSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
	s.MgoSuite.SetUpTest(c)
}

func (s *BaseMgoSuite) TearDownTest(c *gc.C) {
	s.MgoSuite.TearDownTest(c)
	s.BaseSuite.TearDownTest(c)
}
