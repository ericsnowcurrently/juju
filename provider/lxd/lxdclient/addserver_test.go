// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxdclient

import (
	"fmt"
	"strings"

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

func (t urlTest) checkValid(c *gc.C) {
	c.Assert(t.err, gc.Equals, "")

	fixed, err := fixURL(t.url)

	if !c.Check(err, jc.ErrorIsNil) {
		return
	}
	c.Logf("   fixed:  %q", fixed)
	c.Check(fixed, gc.Equals, t.expected)
}

var typicalURLTests = []urlTest{{
	url:      "a.b.c",
	expected: "https://a.b.c:8443",
}, {
	url:      "https://a.b.c/",
	expected: "https://a.b.c:8443",
}, {
	url:      "https://a.b.c:1234/",
	expected: "https://a.b.c:1234",
}, {
	url:      "https://a.b.c/x/y/z",
	expected: "https://a.b.c:8443/x/y/z",
}, {
	url:      "1.2.3.4",
	expected: "https://1.2.3.4:8443",
}}

func (s *fixURLSuite) TestFixURLTypical(c *gc.C) {
	for i, test := range typicalURLTests {
		c.Logf("test %d: checking %q", i, test.url)
		test.checkValid(c)
	}
}

type urlTestSpec struct {
	host         string
	path         string
	port         string
	expectedHost string
}

func (ts urlTestSpec) test(scheme, sep, trailer string) urlTest {
	if scheme != "" {
		scheme += ":"
	}
	host := ts.host
	if ts.port != "" {
		host += ":" + ts.port
	}
	expectedHost := ts.expectedHost
	if expectedHost == "" {
		expectedHost = ts.host
	}
	expectedPort := ts.port
	if expectedPort == "" {
		expectedPort = "8443"
	}
	return urlTest{
		url:      fmt.Sprintf("%s%s%s%s%s", scheme, sep, host, ts.path, trailer),
		expected: fmt.Sprintf("https://%s:%s%s", expectedHost, expectedPort, ts.path),
	}
}

func (ts urlTestSpec) tests() []urlTest {
	var tests []urlTest
	for _, trailer := range []string{"", "/"} {
		tests = append(tests, ts.test("", "", trailer))
		for _, scheme := range []string{"", "https", "http"} {
			tests = append(tests, ts.test(scheme, "//", trailer))
		}
	}
	return tests
}

var validURLHosts = map[string]string{
	"a.b.c":                    "",
	"10.0.0.1":                 "",
	"127.0.0.1":                "",
	"2001:db8::ff00:42:8329":   "[2001:db8::ff00:42:8329]",
	"[2001:db8::ff00:42:8329]": "",
	"::1":   "[::1]",
	"[::1]": "",
}

func (s *fixURLSuite) TestFixURLAllValid(c *gc.C) {
	var specs []urlTestSpec
	for host, expectedHost := range validURLHosts {
		for _, path := range []string{"", "/x/y/z"} {
			specs = append(specs, urlTestSpec{host, path, "", expectedHost})
			if !strings.Contains(host, ":") || strings.HasPrefix(host, "[") {
				specs = append(specs, urlTestSpec{host, path, "1234", expectedHost})
			}
		}
	}
	for i, spec := range specs {
		for j, test := range spec.tests() {
			c.Logf("test %d.%d: checking %q", i, j, test.url)
			test.checkValid(c)
		}
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
