// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"time"

	"github.com/juju/utils/filestorage"

	"github.com/juju/juju/version"
)

const checksumFormat = "SHA-1, base64 encoded"

//---------------------------
// origin

// Origin identifies where a backup archive came from.
type Origin struct {
	environment string
	machine     string
	hostname    string
	version     version.Number
}

func NewOrigin(env, machine, hostname string, vers version.Number) *Origin {
	origin := Origin{
		environment: env,
		machine:     machine,
		hostname:    hostname,
		version:     vers,
	}
	return &origin
}

// Environment is the ID for the backed-up environment.
func (o *Origin) Environment() string {
	return o.environment
}

// Machine is the ID of the state "machine" that ran the backup.
func (o *Origin) Machine() string {
	return o.machine
}

// Hostname is where the backup happened.
func (o *Origin) Hostname() string {
	return o.hostname
}

// Version is the version of juju used to produce the backup.
func (o *Origin) Version() version.Number {
	return o.version
}

//---------------------------
// metadata

// Metadata contains the metadata for a single state backup archive.
type Metadata struct {
	filestorage.FileMetadata
	finished *time.Time
	origin   Origin
	notes    string // not required
}

// NewMetadata returns a new Metadata for a state backup archive.  The
// current date/time is used for the timestamp and the default checksum
// format is used.  ID is not set.  That is left up to the persistence
// layer.  Archived is set as false.  "notes" may be empty, but
// everything else should be provided.
func NewMetadata(
	size int64, checksum string, origin Origin, notes string,
) *Metadata {
	return NewMetadataFull(
		size,
		checksum,
		origin,
		notes,
		checksumFormat,
		time.Now().UTC(),
		nil,
	)
}

func NewMetadataFull(
	size int64, checksum string, origin Origin,
	notes, checksumFormat string, started time.Time, finished *time.Time,
) *Metadata {
	raw := filestorage.NewMetadata(
		size, checksum, checksumFormat, &started)
	fmeta, ok := raw.(*filestorage.FileMetadata)
	if !ok {
		panic("filestorage.NewMetadata should have returned a FileMetadata")
	}
	metadata := Metadata{*fmeta, finished, origin, notes}
	return &metadata
}

// Started records when the backup process started for the archive.
func (m *Metadata) Started() time.Time {
	return m.Timestamp()
}

// Finished records when the backup process finished for the archive.
func (m *Metadata) Finished() *time.Time {
	return m.finished
}

// Origin identifies where the backup was created.
func (m *Metadata) Origin() Origin {
	return m.origin
}

// Notes contains user-supplied annotations for the archive, if any.
func (m *Metadata) Notes() string {
	return m.notes
}
