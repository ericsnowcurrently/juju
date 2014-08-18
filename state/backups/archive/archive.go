// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package archive

import (
	"path/filepath"
)

const (
	contentDir  = "juju-backup"
	filesBundle = "root.tar"
	dbDumpDir   = "dump"
)

// Archive is an abstraction of a backup archive and it's associated
// contents.
type Archive interface {
	// Filename is the path to the archilve file.
	Filename() string
	// RootDir is the path to the root directory for the unpacked contents.
	RootDir() string
	// ContentDir is the path to the directory within the archive containing
	// all the contents.
	ContentDir() string
	// FilesBundle is the path to the tar file inside the archive containing
	// all the state-related files gathered in by the backup machinery.
	FilesBundle() string
	// DBDumpDir is the path to the directory within the archive contents
	// that contains all the files dumped from the juju state database.
	DBDumpDir() string
}

type archive struct {
	filename string
	rootDir  string
}

// NewRootedArchive returns a new archive summary for the filename and
// root directory for the unpacked contents.
func NewRootedArchive(filename, rootDir string) *archive {
	ar := archive{
		filename: filename,
		rootDir:  rootDir,
	}
	return &ar
}

// NewArchive returns a new archive summary for the filename.
func NewArchive(filename string) *archive {
	ar := archive{
		filename: filename,
	}
	return &ar
}

func (ar *archive) Filename() string {
	return ar.filename
}

func (ar *archive) RootDir() string {
	return ar.rootDir
}

func (ar *archive) ContentDir() string {
	return filepath.Join(ar.rootDir, contentDir)
}

func (ar *archive) FilesBundle() string {
	return filepath.Join(ar.ContentDir(), filesBundle)
}

func (ar *archive) DBDumpDir() string {
	return filepath.Join(ar.ContentDir(), dbDumpDir)
}
