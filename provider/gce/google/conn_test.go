// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package google_test

import (
	"regexp"

	"code.google.com/p/google-api-go-client/compute/v1"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/provider/gce/google"
)

type connSuite struct {
	google.BaseSuite

	conn *google.Connection
}

var _ = gc.Suite(&connSuite{})

func (s *connSuite) TestConnectionConnect(c *gc.C) {
	google.SetRawConn(s.Conn, nil)
	service := &compute.Service{}
	s.PatchValue(google.NewRawConnection, func(auth google.Auth) (*compute.Service, error) {
		return service, nil
	})

	err := s.Conn.Connect(s.Auth)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(google.ExposeRawService(s.Conn), gc.Equals, service)
}

func (s *connSuite) TestConnectionConnectAlreadyConnected(c *gc.C) {
	err := s.Conn.Connect(s.Auth)

	c.Check(err, gc.ErrorMatches, regexp.QuoteMeta(`connect() failed (already connected)`))
}

func (s *connSuite) TestConnectionVerifyCredentials(c *gc.C) {
	s.FakeConn.Project = &compute.Project{}
	err := s.Conn.VerifyCredentials()

	c.Check(err, jc.ErrorIsNil)
}

func (s *connSuite) TestConnectionVerifyCredentialsAPI(c *gc.C) {
	s.FakeConn.Project = &compute.Project{}
	err := s.Conn.VerifyCredentials()
	c.Assert(err, jc.ErrorIsNil)

	c.Check(s.FakeConn.Calls, gc.HasLen, 1)
	c.Check(s.FakeConn.Calls[0].FuncName, gc.Equals, "GetProject")
	c.Check(s.FakeConn.Calls[0].ProjectID, gc.Equals, "spam")
}

func (s *connSuite) TestConnectionVerifyCredentialsInvalid(c *gc.C) {
	s.FakeConn.Err = errors.New("retrieving auth token for user@mail.com: Invalid Key")
	err := s.Conn.VerifyCredentials()

	c.Check(err, gc.ErrorMatches, `retrieving auth token for user@mail.com: Invalid Key`)
}

func (s *connSuite) TestConnectionAvailabilityZones(c *gc.C) {
	s.FakeConn.Zones = []*compute.Zone{{
		Name:   "a-zone",
		Status: google.StatusUp,
	}}

	azs, err := s.Conn.AvailabilityZones("a")
	c.Check(err, gc.IsNil)

	c.Check(len(azs), gc.Equals, 1)
	c.Check(azs[0].Name(), gc.Equals, "a-zone")
	c.Check(azs[0].Status(), gc.Equals, google.StatusUp)
}

func (s *connSuite) TestConnectionAvailabilityZonesAPI(c *gc.C) {
	_, err := s.Conn.AvailabilityZones("a")
	c.Assert(err, gc.IsNil)

	c.Check(s.FakeConn.Calls, gc.HasLen, 1)
	c.Check(s.FakeConn.Calls[0].FuncName, gc.Equals, "ListAvailabilityZones")
	c.Check(s.FakeConn.Calls[0].ProjectID, gc.Equals, "spam")
	c.Check(s.FakeConn.Calls[0].Region, gc.Equals, "a")
}

func (s *connSuite) TestConnectionAvailabilityZonesErr(c *gc.C) {
	s.FakeConn.Err = errors.New("<unknown>")

	_, err := s.Conn.AvailabilityZones("a")

	c.Check(err, gc.ErrorMatches, "<unknown>")
}
