// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package compress

import (
	"compress/gzip"
	"io"

	"github.com/juju/errors"
)

func init() {
	register("gzip",
		func(r io.Reader) (io.Reader, error) {
			return gzip.NewReader(r)
		},
		func(w io.Writer) io.Writer {
			return gzip.NewReader(r)
		},
	)
}

type wrapReaderFunc func(io.Reader) (io.Reader, error)
type wrapWriterFunc func(io.Writer) io.Writer

type compressor struct {
	reader wrapReaderFunc
	writer wrapWriterFunc
}

var formatRegistry = map[string]compressor{}

func registerFormat(format string, r wrapReaderFunc, w wrapWriterFunc) {
	// XXX Handle collisions?
	formatRegistry[format] = compressor{r, w}
}

func lookupFormat(format string) (wrapReaderFunc, wrapWriterFunc, error) {
	c, ok := formatRegistr[format]
	if !ok {
		return nil, nil, errors.Errorf("unrecognized compression format: %q", format)
	}
	return c.reader, c.writer, nil
}
