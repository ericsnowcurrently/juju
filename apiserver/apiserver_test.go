// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	//"net/http"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver"
	coretesting "github.com/juju/juju/testing"
)

type ServerSuite struct {
	coretesting.BaseSuite
}

var _ = gc.Suite(&ServerSuite{})

func (s *ServerSuite) TestHTTPEndpointsDefault(c *gc.C) {
	var server apiserver.Server

	endpoints := server.HTTPEndpoints()

	var patterns []string
	for _, ep := range endpoints {
		patterns = append(patterns, ep.Pattern)
	}
	c.Check(patterns, jc.DeepEquals, []string{
		"/environment/:envuuid/log",
		"/environment/:envuuid/charms",
		"/environment/:envuuid/tools",
		"/environment/:envuuid/tools/:version",
		"/environment/:envuuid/backups",
		"/environment/:envuuid/images/:kind/:series/:arch/:filename",
		"/environment/:envuuid/api",
	})
}

func (s *ServerSuite) TestHTTPEndpointsDBLog(c *gc.C) {
	s.SetFeatureFlags("db-log")
	var server apiserver.Server

	endpoints := server.HTTPEndpoints()

	var patterns []string
	for _, ep := range endpoints {
		patterns = append(patterns, ep.Pattern)
	}
	c.Check(patterns, jc.DeepEquals, []string{
		"/environment/:envuuid/logsink",
		"/environment/:envuuid/log",
		"/environment/:envuuid/charms",
		"/environment/:envuuid/tools",
		"/environment/:envuuid/tools/:version",
		"/environment/:envuuid/backups",
		"/environment/:envuuid/images/:kind/:series/:arch/:filename",
		"/environment/:envuuid/api",
	})
}
