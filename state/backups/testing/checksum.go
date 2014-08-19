// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"crypto/sha1"
	"encoding/base64"
	"io"
	"os"

	gc "launchpad.net/gocheck"
)

func shaSumFile(c *gc.C, filename string) string {
	f, err := os.Open(filename)
	c.Assert(err, gc.IsNil)
	defer f.Close()

	shahash := sha1.New()

	_, err = io.Copy(shahash, f)
	c.Assert(err, gc.IsNil)

	return base64.StdEncoding.EncodeToString(shahash.Sum(nil))
}
