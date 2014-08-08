// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// backups contains all the stand-alone backup-related functionality for
// juju state.
package backups

import (
	"io"
	"time"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/utils/filestorage"

	"github.com/juju/juju/state/backups/config"
)

var logger = loggo.GetLogger("juju.state.backup")

// Backups is an abstraction around all juju backup-related functionality.
type Backups interface {
	// Create creates and stores a new juju backup archive and returns
	// its associated metadata.
	Create(origin *Origin, notes string) (*Metadata, error)
	// Get returns the metadata and archive file associated with the ID.
	Get(id string) (*Metadata, io.ReadCloser, error)
}

type backups struct {
	config  config.BackupsConfig
	storage filestorage.FileStorage
}

// NewBackups returns a new Backups value using the provided DB info and
// file storage.
func NewBackups(
	config config.BackupsConfig, stor filestorage.FileStorage,
) Backups {
	b := backups{
		config:  config,
		storage: stor,
	}
	return &b
}

// Create creates and stores a new juju backup archive and returns
// its associated metadata.
func (b *backups) Create(origin *Origin, notes string) (*Metadata, error) {
	// Prep the metadata.
	meta := NewMetadata("", 0, *origin, notes)

	// Create the archive.
	archive, err := create(b.config, meta)
	if err != nil {
		return nil, errors.Annotate(err, "error creating backup archive")
	}
	defer archive.Close()

	// Store the archive.
	finished := time.Now().UTC()
	meta.finished = &finished
	_, err = b.storage.Add(meta, archive)
	if err != nil {
		return nil, errors.Annotate(err, "error storing backup archive")
	}

	return meta, nil
}

// Get returns the metadata and archive file associated with the ID.
func (b *backups) Get(id string) (*Metadata, io.ReadCloser, error) {
	rawmeta, archive, err := b.storage.Get(id)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	meta, ok := rawmeta.(*Metadata)
	if !ok {
		panic("did not get a backup.Metadata value from storage")
	}

	return meta, archive, nil
}
