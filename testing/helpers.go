// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"bytes"
	"io"
	"io/ioutil"
)

// NewCloseableBufferStrung returns a closeable wrapper around a
// bytes.Buffer, allowing the buffer to be used as an io.ReadCloser.
func NewCloseableBufferString(data string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBufferString(data))
}
