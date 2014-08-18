// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package create

import (
	"io"
	"os"
	"time"

	"github.com/juju/errors"

	"github.com/juju/juju/state/backups/archive"
)

// Result is the result of building a backup archive.
type Result struct {
	ArchiveFile io.ReadCloser
	Size        int64
	Checksum    string
	Timestamp   time.Time
}

// NewResult returns the result of building a backup archive.
func NewResult(ar archive.Archive, checksum string) (*Result, error) {
	// Re-open the archive for reading.
	archiveFile, err := os.Open(ar.Filename())
	if err != nil {
		return nil, errors.Annotate(err, "error opening temp archive")
	}

	stat, err := archiveFile.Stat()
	if err != nil {
		return nil, errors.Annotate(err, "error reading file info")
	}
	size := stat.Size()

	result := Result{
		ArchiveFile: archiveFile,
		Size:        size,
		Checksum:    checksum,
		Timestamp:   time.Now().UTC(),
	}
	return &result, nil
}
