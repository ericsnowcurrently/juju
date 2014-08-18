// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package create

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/juju/errors"

	"github.com/juju/juju/state/backups/archive"
)

const (
	rootDirPrefix = "jujuBackup-"
	archiveName   = "backup.tar.gz"
)

type workspace struct {
	archive archive.Archive

	archiveFile io.WriteCloser
	bundleFile  io.WriteCloser
}

func newWorkspace() (*workspace, error) {
	// Create the root workspace directory.
	rootDir, err := ioutil.TempDir("", rootDirPrefix)
	if err != nil {
		return nil, errors.Annotate(err, "error creating root temp directory")
	}

	// Create the archive.
	filename := filepath.Join(rootDir, archiveName)
	ar := archive.NewRootedArchive(filename, rootDir)

	// Create the rest of the directories.  We go with user-only
	// permissions on principle; the directories are short-lived so in
	// practice it shouldn't matter much.
	err = os.MkdirAll(ar.DBDumpDir(), os.FileMode(0755))
	if err != nil {
		return nil, errors.Annotate(err, "error creating temp directories")
	}

	// Create the workspace.
	ws := workspace{
		archive: ar,
	}
	return &ws, nil
}

func (ws *workspace) createArchiveFile() error {
	archiveFile, err := os.Create(ws.archive.Filename())
	if err != nil {
		return errors.Annotate(err, "error creating backup archive file")
	}

	ws.archiveFile = archiveFile
	return nil
}

func (ws *workspace) createBundleFile() error {
	bundleFile, err := os.Create(ws.archive.FilesBundle())
	if err != nil {
		return errors.Annotate(err, "error creating files bundle archive")
	}

	ws.bundleFile = bundleFile
	return nil
}

func (ws *workspace) closeArchiveFile() error {
	if ws.archiveFile != nil {
		if err := ws.archiveFile.Close(); err != nil {
			return errors.Annotate(err, "error closing archive file")
		}
		ws.archiveFile = nil
	}
	return nil
}

func (ws *workspace) closeBundleFile() error {
	if ws.bundleFile != nil {
		if err := ws.bundleFile.Close(); err != nil {
			return errors.Annotate(err, "error closing bundle file")
		}
		ws.bundleFile = nil
	}
	return nil
}

func (ws *workspace) Close() error {
	var failed bool

	if err := ws.closeArchiveFile(); err != nil {
		logger.Errorf(err.Error())
		failed = true
	}
	if err := ws.closeBundleFile(); err != nil {
		logger.Errorf(err.Error())
		failed = true
	}

	if failed {
		return errors.New("error closing backup builder (see logs)")
	}

	return nil
}

func (ws *workspace) cleanUp() error {
	var failed bool

	if err := ws.Close(); err != nil {
		failed = true
	}

	if ws.archive != nil {
		err := os.RemoveAll(ws.archive.RootDir())
		if err != nil {
			logger.Errorf("error deleting temp directory: %v", err)
			failed = true
		}
		ws.archive = nil
	}

	if failed {
		return errors.New("error cleaning up backup builder (see logs)")
	}

	return nil
}
