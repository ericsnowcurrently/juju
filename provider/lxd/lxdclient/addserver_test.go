// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxdclient

import (
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(&fixURLSuite{})

type fixURLSuite struct {
	testing.IsolationSuite
}

func (s *fixURLSuite) TestFixURLLocal(c *gc.C) {
	fixed, err := fixURL("")
	c.Assert(err, jc.ErrorIsNil)

	c.Check(fixed, gc.Equals, "")
}

type urlTest struct {
	url      string
	expected string
	err      string
}

var typicalURLTests = []urlTest{{
	url:      "a.b.c",
	expected: "https://a.b.c:8443",
}, {
	url:      "https://a.b.c",
	expected: "https://a.b.c:8443",
}, {
	url:      "https://a.b.c/",
	expected: "https://a.b.c:8443",
}, {
	url:      "a.b.c:1234",
	expected: "https://a.b.c:1234",
}, {
	url:      "https://a.b.c:1234",
	expected: "https://a.b.c:1234",
}, {
	url:      "https://a.b.c:1234/",
	expected: "https://a.b.c:1234",
}, {
	url:      "a.b.c/x/y/z",
	expected: "https://a.b.c:8443/x/y/z",
}, {
	url:      "https://a.b.c/x/y/z",
	expected: "https://a.b.c:8443/x/y/z",
}, {
	url:      "https://a.b.c:1234/x/y/z",
	expected: "https://a.b.c:1234/x/y/z",
}, {
	url:      "http://a.b.c",
	expected: "https://a.b.c:8443",
}, {
	url:      "1.2.3.4",
	expected: "https://1.2.3.4:8443",
}, {
	url:      "1.2.3.4:1234",
	expected: "https://1.2.3.4:1234",
}, {
	url:      "https://1.2.3.4",
	expected: "https://1.2.3.4:8443",
}, {
	url:      "127.0.0.1",
	expected: "https://127.0.0.1:8443",
}, {
	url:      "2001:db8::ff00:42:8329",
	expected: "https://[2001:db8::ff00:42:8329]:8443",
}, {
	url:      "2001:db8::ff00:42:8329/",
	expected: "https://[2001:db8::ff00:42:8329]:8443",
}, {
	url:      "[2001:db8::ff00:42:8329]:1234",
	expected: "https://[2001:db8::ff00:42:8329]:1234",
}, {
	url:      "[2001:db8::ff00:42:8329]:1234/",
	expected: "https://[2001:db8::ff00:42:8329]:1234",
}, {
	url:      "https://2001:db8::ff00:42:8329",
	expected: "https://[2001:db8::ff00:42:8329]:8443",
}, {
	url:      "https://2001:db8::ff00:42:8329/",
	expected: "https://[2001:db8::ff00:42:8329]:8443",
}, {
	url:      "https://[2001:db8::ff00:42:8329]:1234",
	expected: "https://[2001:db8::ff00:42:8329]:1234",
}, {
	url:      "https://[2001:db8::ff00:42:8329]:1234/",
	expected: "https://[2001:db8::ff00:42:8329]:1234",
}, {
	url:      "2001:db8::ff00:42:8329/x/y/z",
	expected: "https://[2001:db8::ff00:42:8329]:8443/x/y/z",
}, {
	url:      "[2001:db8::ff00:42:8329]:1234/x/y/z",
	expected: "https://[2001:db8::ff00:42:8329]:1234/x/y/z",
}, {
	url:      "https://2001:db8::ff00:42:8329/x/y/z",
	expected: "https://[2001:db8::ff00:42:8329]:8443/x/y/z",
}, {
	url:      "https://[2001:db8::ff00:42:8329]:1234/x/y/z",
	expected: "https://[2001:db8::ff00:42:8329]:1234/x/y/z",
}, {
	url:      "::1",
	expected: "https://[::1]:8443",
}, {
	url:      "::1/",
	expected: "https://[::1]:8443",
}, {
	url:      "::1/x/y/z",
	expected: "https://[::1]:8443/x/y/z",
}, {
	url:      "[::1]:1234",
	expected: "https://[::1]:1234",
}, {
	url:      "[::1]:1234/",
	expected: "https://[::1]:1234",
}, {
	url:      "[::1]/x/y/z",
	expected: "https://[::1]:8443/x/y/z",
}, {
	url:      "[::1]:1234/x/y/z",
	expected: "https://[::1]:1234/x/y/z",
}, {
	url:      "https://::1",
	expected: "https://[::1]:8443",
}, {
	url:      "https://::1/",
	expected: "https://[::1]:8443",
}, {
	url:      "::1/x/y/z",
	expected: "https://[::1]:8443/x/y/z",
}, {
	url:      "https://::1/x/y/z",
	expected: "https://[::1]:8443/x/y/z",
}}

func (s *fixURLSuite) TestFixURLTypical(c *gc.C) {
	for i, test := range typicalURLTests {
		c.Logf("test %d: checking %q", i, test.url)
		c.Assert(test.err, gc.Equals, "")

		fixed, err := fixURL(test.url)

		if !c.Check(err, jc.ErrorIsNil) {
			continue
		}
		c.Check(fixed, gc.Equals, test.expected)
	}
}

// TODO(ericsnow) Add failure tests for bad domain names
// and IP addresses (v4/v6).

var failureURLTests = []urlTest{{
	// a malformed URL
	url: ":a.b.c",
	err: `.*`,
}, {
	url: "unix://",
	err: `.*unix socket URLs not supported.*`,
}, {
	url: "unix:///x/y/z",
	err: `.*unix socket URLs not supported.*`,
}, {
	url: "unix:/x/y/z",
	err: `.*unix socket URLs not supported.*`,
}, {
	url: "/x/y/z",
	err: `.*unix socket URLs not supported.*`,
}, {
	url: "https://a.b.c:xyz",
	err: `.*invalid port.*`,
}, {
	url: "https://a.b.c:0",
	err: `.*invalid port.*`,
}, {
	url: "https://a.b.c:99999",
	err: `.*invalid port.*`,
}, {
	url: "spam://a.b.c",
	err: `.*unsupported URL scheme.*`,
}, {
	url: "https://a.b.c?d=e",
	err: `.*URL queries not supported.*`,
}, {
	url: "https://a.b.c#d",
	err: `.*URL fragments not supported.*`,
}}

func (s *fixURLSuite) TestFixURLFailures(c *gc.C) {
	for i, test := range failureURLTests {
		c.Logf("test %d: checking %q", i, test.url)
		c.Assert(test.err, gc.Not(gc.Equals), "")

		_, err := fixURL(test.url)

		c.Check(err, gc.ErrorMatches, test.err)
	}
}
