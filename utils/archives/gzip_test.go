// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package archives_test

import (
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/utils/archives"
)

func (s *ArchivesSuite) TestGzipData(c *gc.C) {
	data := "spam"
	compressed, err := archives.GzipData(data)

	c.Assert(err, gc.IsNil)
	expected := "\x1f\x8b\x08\x00\x00\x09\x6e\x88" +
		"\x00\xff\x2a\x2e\x48\xcc\x05\x04" +
		"\x00\x00\xff\xff\x3d\xff\xda\x43" +
		"\x04\x00\x00\x00"
	c.Assert(compressed, gc.Equals, expected)
}
