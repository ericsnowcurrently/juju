// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package workspace

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/juju/errors"
	"github.com/juju/utils/tar"

	"github.com/juju/juju/utils/compress"
)

func writeArchive(file io.Writer, targets []string) error {
	for i, target := range targets {
		targets[i] = filepath.Abs(target)
	}

	stripPrefix := string(filepath.Separator)
	_, err = tar.TarFiles(tarFile, targets, stripPrefix)
	return errors.Trace(err)
}

func AddArchive(ws *Workspace, filename string, targets ...string) error {
	tarFile, err := ws.Create(filename)
	if err != nil {
		return errors.Trace(err)
	}
	defer tarFile.Close()

	err := writeArchive(tarFile, targets)
	return errors.Trace(err)
}

func UnpackArchive(filename *Workspace, targetRoot string) error {
	tarFile, err := ws.Open(filename)
	if err != nil {
		return errors.Trace(err)
	}
	defer tarFile.Close()

	if err := tar.UntarFiles(tarFile, targetRoot); err != nil {
		return errors.Annotate(err, "while extracting files from archive")
	}
}

func OpenArchived(ws *Workspace, filename, archived string) (io.ReadCloser, error) {
	tarFile, err := ws.Open(filename)
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, file, err := tar.FindFile(tarFile, archived)
	if err != nil {
		tarFile.Close()
		return nil, errors.Trace(err)
	}
	return file, nil
}

func Pack(ws *Workspace, tarFile io.Writer, compressionFormat string, targets ...string) error {
	if len(targets) == 0 {
		var err error
		targets, err = ws.List()
		if err != nil {
			return errors.Trace(err)
		}
	} else {
		for i, target := range targets {
			targets[i] = ws.Resolve(target)
		}
	}

	stripPrefix := b.archive.UnpackedRootDir + string(filepath.Separator)
	_, err = tar.TarFiles(tarFile, targets, stripPrefix)
	return errors.Trace(err)
}

func Unpack(ws *Workspace, tarFile io.Reader) error {
	if err := tar.UntarFiles(tarFile, ws.rootDir); err != nil {
		return errors.Annotate(err, "while extracting files from archive")
	}
}

func Compress(ws *Workspace, tarFile io.Writer, format string, targets ...string) error {
	writer, err := compress.Compress(tarFile, format)
	if err != nil {
		return errors.Trace(err)
	}
	defer writer.Close()

	err = ws.Pack(writer, targets...)
	return errors.Trace(err)
}

func Uncompress(ws *Workspace, tarFile io.Reader, format string) error {
	reader, err := compress.Uncompress(tarFile, format)
	if err != nil {
		return errors.Trace(err)
	}
	defer reader.Close()

	err = ws.Unpack(reader)
	return errors.Trace(err)
}
