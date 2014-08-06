// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"testing"

	gitjujutesting "github.com/juju/testing"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/environmentserver/authentication"
	"github.com/juju/juju/mongo"
)

// MgoTestPackage should be called to register the tests for any package
// that requires a secure connection to a MongoDB server.
func MgoTestPackage(t *testing.T) {
	gitjujutesting.MgoTestPackage(t, Certs)
}

// MgoSuite is a core-specific wrapper around testing.MgoSuite.
type MgoSuite struct {
	gitjujutesting.MgoSuite
}

func (s *MgoSuite) Server() *gitjujutesting.MgoInstance {
	// XXX(ericsnow) Eliminate this global state!
	return gitjujutesting.MgoServer
}

// BaseMgoSuite adds BaseSuite to the local MgoSuite.
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

// FakeHomeMgoSuite adds FakeJujuHomeSuite to the local MgoSuite.
type FakeHomeMgoSuite struct {
	FakeJujuHomeSuite
	MgoSuite
}

func (s *FakeHomeMgoSuite) SetUpSuite(c *gc.C) {
	s.FakeJujuHomeSuite.SetUpSuite(c)
	s.MgoSuite.SetUpSuite(c)
}

func (s *FakeHomeMgoSuite) TearDownSuite(c *gc.C) {
	s.MgoSuite.TearDownSuite(c)
	s.FakeJujuHomeSuite.TearDownSuite(c)
}

func (s *FakeHomeMgoSuite) SetUpTest(c *gc.C) {
	s.FakeJujuHomeSuite.SetUpTest(c)
	s.MgoSuite.SetUpTest(c)
}

func (s *FakeHomeMgoSuite) TearDownTest(c *gc.C) {
	s.MgoSuite.TearDownTest(c)
	s.FakeJujuHomeSuite.TearDownTest(c)
}
