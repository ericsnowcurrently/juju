// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package archive

import (
	"io"
	"os"
	"path/filepath"

	"github.com/juju/errors"

	"github.com/juju/juju/state/backups/metadata"
	"github.com/juju/juju/utils/workspace"
)

// Workspace is a wrapper around backup archive info that has a concrete
// root directory and an archive unpacked in it.
type Workspace struct {
	workspace.Workspace
	Archive
}

func newWorkspace(filename string) (*Workspace, error) {
	base, err := workspace.NewWorkspace("juju-backups-")
	if err != nil {
		return nil, errors.Trace(err)
	}

	ar := NewArchive(filename, base.RootDir())

	ws := Workspace{
		Workspace: *base,
		Archive:   *ar,
	}
	return &ws, nil
}

// NewWorkspace returns a new archive workspace with a new workspace dir.
func NewWorkspace(filename string) (*Workspace, error) {
	ws, err := newWorkspace(filename)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if archiveFile, err := os.Open(ws.Filename); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Trace(err)
		}
	} else {
		if err := workspace.Uncompress(ws, archiveFile, "gzip"); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return ws, nil
}

// NewWorkspace returns a new archive workspace with a new workspace dir
// populated from the archive file.
func NewWorkspaceFromFile(archiveFile io.Reader) (*Workspace, error) {
	ws, err := newWorkspace("")
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = workspace.Uncompress(ws, archiveFile, "gzip")
	return ws, errors.Trace(err)
}

// UnpackFiles unpacks the archived files bundle into the targeted dir.
func (ws *Workspace) UnpackFiles(targetRoot string) error {
	err := workspace.UnpackArchive(ws.FilesBundle(), targetRoot)
	return errors.Trace(err)
}

// OpenFile returns an open ReadCloser for the corresponding file in
// the archived files bundle.
func (ws *Workspace) OpenFile(filename string) (io.Reader, error) {
	if filepath.IsAbs(filename) {
		return nil, errors.Errorf("filename must be relative, got %q", filename)
	}
	file, err := workspace.OpenArchived(ws.FilesBundle())
	return file, errors.Trace(err)
}

// Metadata returns the metadata derived from the JSON file in the archive.
func (ws *Workspace) Metadata() (*metadata.Metadata, error) {
	metaFile, err := os.Open(ws.MetadataFile())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer metaFile.Close()

	meta, err := metadata.NewFromJSONBuffer(metaFile)
	return meta, errors.Trace(err)
}
