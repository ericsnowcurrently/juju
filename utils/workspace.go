// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/errors"
)

const workspacePrefix = "juju-workspace-"

// Workspace is a wrapper around a temporary directory.
type Workspace interface {
	// RootDir is the path to the directory at the root of the workspace.
	RootDir() string
	// Resolve returns the path, resolved relative to the workspace.
	Resolve(path string) string
	// Sub returns a new sub-workspace, making the directory if needed.
	Sub(path string) (Workspace, error)
	// MkDirs creates a new directory at the path and returns a workspace
	// that wraps the new directory.
	MkDirs(path string, mode os.FileMode) (string, error)
	// Touch creates a new empty file in the root directory of the
	// workspace.  If the filename is blank, a random (available) one is
	// generated.  The name of the new file is returned.
	Touch(filename string) (string, error)
	// Create creates the file (it must not exist already).  If the
	// filename is blank, a random (available) one is generated.  The
	// new file and its name are returned.
	Create(filename string) (string, *os.File, error)
	// Open opens the file for reading.
	Open(filename string) (*os.File, error)
	// CleanUp removes the workspace root directory.
	CleanUp() error
}

type subWSFunc func(string) (Workspace, error)

type WorkspaceTree struct {
	rootDir string
	relPath string

	subs map[string]*WorkspaceTree
}

// Ensure the interface is implemented.
var _ = Workspace((*WorkspaceTree)(nil))

// NewWorkspace returns a new workspace.
func NewWorkspace() (*WorkspaceTree, error) {
	rootDir, err := ioutil.TempDir("", workspacePrefix)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ws := NewRootedWorkspace(rootDir)
	return ws, nil
}

// NewRootedWorkspace returns a new workspace with an explicit root.
func NewRootedWorkspace(rootDir string) *WorkspaceTree {
	ws := WorkspaceTree{
		rootDir: rootDir,
	}
	return &ws
}

func (ws *WorkspaceTree) RootDir() string {
	return ws.Resolve("")
}

func (ws *WorkspaceTree) Resolve(path string) string {
	return filepath.Join(ws.rootDir, ws.relPath, path)
}

func (ws *WorkspaceTree) Sub(path string) (Workspace, error) {
	path, err := ws.MkDirs(path, 0777)
	if err != nil {
		return nil, errors.Trace(err)
	}
	relPath := filepath.Join(ws.relPath, path)

	if ws.subs != nil {
		// Return the cached one, if any.
		if sub, found := ws.subs[relPath]; found {
			return sub, nil
		}
	} else {
		ws.subs = make(map[string]*WorkspaceTree)
	}

	sub := NewRootedWorkspace(ws.rootDir)
	sub.relPath = relPath
	// All workspaces with the same root share the same subs.
	sub.subs = ws.subs

	// Cache the sub-workspace.
	ws.subs[relPath] = sub

	return sub, nil
}

func (ws *WorkspaceTree) tempDir(prefix string) (string, error) {
	tempDir, err := ioutil.TempDir(ws.RootDir(), prefix)
	if err != nil {
		return "", errors.Trace(err)
	}

	sep := string(filepath.Separator)
	path := strings.TrimPrefix(tempDir, ws.RootDir()+sep)
	return path, nil
}

func (ws *WorkspaceTree) MkDirs(path string, mode os.FileMode) (string, error) {
	if path == "" {
		path, err := ws.tempDir("")
		if err != nil {
			return "", errors.Trace(err)
		}
		return path, nil
	}

	err := os.MkdirAll(ws.Resolve(path), mode)
	if err != nil {
		return "", errors.Trace(err)
	}

	return path, nil
}

func (ws *WorkspaceTree) tempFile(prefix string) (string, *os.File, error) {
	tempfile, err := ioutil.TempFile(ws.RootDir(), prefix)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	filename := filepath.Base(tempfile.Name())
	return filename, tempfile, nil
}

func (ws *WorkspaceTree) Touch(filename string) (string, error) {
	var file *os.File

	if filename == "" {
		var err error
		filename, file, err = ws.tempFile("")
		if err != nil {
			return "", errors.Trace(err)
		}
	} else {
		path := ws.Resolve(filename)

		now := time.Now()
		err := os.Chtimes(path, now, now)
		if err == nil {
			// Short-circuit for existing files.
			return filename, nil
		}

		if !os.IsNotExist(err) {
			return "", errors.Trace(err)
		}

		file, err = os.Create(path)
		if err != nil {
			return "", errors.Trace(err)
		}
	}

	err := file.Close()
	if err != nil {
		return "", errors.Trace(err)
	}

	return filename, nil
}

func (ws *WorkspaceTree) Create(filename string) (string, *os.File, error) {
	var file *os.File

	if filename == "" {
		var err error
		filename, file, err = ws.tempFile("")
		if err != nil {
			return "", nil, errors.Trace(err)
		}
	} else {
		var err error
		path := ws.Resolve(filename)

		flags := os.O_RDWR | os.O_CREATE | os.O_EXCL
		file, err = os.OpenFile(path, flags, 0666)
		if err != nil {
			return "", nil, errors.Trace(err)
		}
	}

	return filename, file, nil
}

func (ws *WorkspaceTree) Open(filename string) (*os.File, error) {
	if filename == "" {
		return nil, errors.New("missing filename")
	}

	path := ws.Resolve(filename)

	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return file, nil
}

func (ws *WorkspaceTree) CleanUp() error {
	err := os.RemoveAll(ws.RootDir())
	if err != nil {
		return errors.Trace(err)
	}

	ws.subs = nil

	return nil
}
