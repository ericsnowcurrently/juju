// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package compress

import (
	"io"

	"github.com/juju/errors"
)

type flusher interface {
	Flush() error
}

type wReseter interface {
	Reset(io.Writer)
}

type CompressedFile struct {
	io.Writer
}

func Compress(writer io.Writer, format string) (*CompressedFile, error) {
	_, wrap, err := lookupFormat(format)
	if err != nil {
		return nil, errors.Trace(err)
	}
	writer = wrap(writer)

	cf := CompressedFile{
		Writer: writer,
	}
	return &cf, nil
}

func (cf *CompressedFile) Flush() error {
	if f, ok := cf.Writer.(flusher); ok {
		err := f.Flush()
		return errors.Trace(err)
	}
	return errors.NotSupportedf("Flush()")
}

func (cf *CompressedFile) Reset(w io.Writer) error {
	if reseter, ok := cf.Writer.(wReseter); ok {
		err := reseter.Reset(w)
		return errors.Trace(err)
	}
	return errors.NotSupportedf("Reset()")
}

func (cf *CompressedFile) Close() error {
	if closer, ok := cf.Writer.(io.Closer); ok {
		err := closer.Close()
		return errors.Trace(err)
	}
	return nil
}
