// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package compress

import (
	"io"

	"github.com/juju/errors"
)

type rReseter interface {
	Reset(io.Reader)
}

type UncompressedFile struct {
	io.Reader
}

func Uncompress(file io.Reader, format string) (*UncompressedFile, error) {
	wrap, _, err := lookupFormat(format)
	if err != nil {
		return nil, errors.Trace(err)
	}
	reader, err = wrap(reader)
	if err != nil {
		return nil, errors.Trace(err)
	}

	uf := CompressedFile{
		Reader: reader,
	}
	return &uf, nil
}

func (uf *UncompressedFile) Reset(r io.Reader) error {
	if reseter, ok := uf.Reader.(rReseter); ok {
		err := reseter.Reset(r)
		return errors.Trace(err)
	}
	return errors.NotSupportedf("Reset()")
}

func (uf *UncompressedFile) Close() error {
	if closer, ok := uf.Reader.(io.Closer); ok {
		err := closer.Close()
		return errors.Trace(err)
	}
	return nil
}
