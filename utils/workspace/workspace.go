// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package workspace

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/juju/errors"
)

// Workspace is a wrapper around backup archive info that has a concrete
// root directory and an archive unpacked in it.
type Workspace struct {
	rootDir string // XXX If not absolute, a CWD might wreak havok.
	owned   bool
	cached  map[string]os.File
}

func NewWorkspace(prefix string) (*Workspace, error) {
	rootDir, err := ioutil.TempDir("", prefix)
	if err != nil {
		return nil, errors.Annotate(err, "while creating workspace dir")
	}

	ws := Workspace{
		rootDir: rootDir,
		owned:   true,
		cached:  make(map[string]os.File),
	}
	return &ws, nil
}

// RootDir returns the directory name of the workspace root.
func (ws *Workspace) RootDir() string {
	return ws.rootDir
}

// Close cleans up the workspace dir.
func (ws *Workspace) Close() error {
	var errs []error
	for filename, file := range ws.cached {
		if err := file.Close(); err != nil {
			errs = append(errs, errors.annotatef(err, "while closing %q", filename))
		}
		delete(ws.cached, filename)
	}

	if ws.owned {
		err := os.RemoveAll(ws.rootDir)
		if err != nil {
			errs = append(errs, errors.Trace(err))
		}
	}

	return NewMultiError(errs)
}

func (ws *Workspace) Resolve(filename string) string {
	if filename == "" {
		return ""
	}
	return filepath.Join(ws.rootDir, filename)
}

func (ws *TarWorkspace) List() ([]string, error) {
	infos, err := ioutil.ReadDir(ws.rootDir)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var list []string
	for _, info := range infos {
		list = append(list, info.Name)
	}
	return list, nil
}

func (ws *Workspace) Create(filename string) (io.ReadWriteCloser, error) {
	filename = ws.Resolve(filename)
	writer, err := os.Create(filename)
	return writer, errors.Trace(err)
}

func (ws *Workspace) Open(filename string) (io.ReadCloser, error) {
	filename = ws.Resolve(filename)
	reader, err := os.Open(filename)
	return reader, errors.Trace(err)
}

func (ws *Workspace) CreateCached(filename string) (io.ReadWriter, error) {
	filename = ws.Resolve(filename)
	if file, ok := ws.cached[filename]; ok {
		// XXX Verify it is readable and writable.
		return file, nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return file, err
	}
	ws.cached[filename] = file
	return file, nil
}

func (ws *Workspace) OpenCached(filename string) (io.Reader, error) {
	filename = ws.Resolve(filename)
	if file, ok := ws.cached[filename]; ok {
		// XXX Verify it is readable.
		return file, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return file, err
	}
	ws.cached[filename] = file
	return file, nil
}
