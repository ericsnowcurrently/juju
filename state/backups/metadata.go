// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/juju/errors"
	"github.com/juju/utils/filestorage"

	"github.com/juju/juju/state"
	"github.com/juju/juju/version"
)

// checksumFormat identifies how to interpret the checksum for a backup
// generated with this version of juju.
const checksumFormat = "SHA-1, base64 encoded"

// Origin identifies where a backup archive came from.  While it is
// more about where and Metadata about what and when, that distinction
// does not merit special consideration.  Instead, Origin exists
// separately from Metadata due to its use as an argument when
// requesting the creation of a new backup.
type Origin struct {
	Environment string
	Machine     string
	Hostname    string
	Version     version.Number
}

func newOrigin() Origin {
	return Origin{
		Version: version.Current.Number,
	}
}

// Metadata contains the metadata for a single state backup archive.
type Metadata struct {
	*filestorage.FileMetadata

	// Started records when the backup was started.
	Started time.Time
	// Finished records when the backup was complete.
	Finished *time.Time
	// Origin identifies where the backup was created.
	Origin Origin
	// Notes is an optional user-supplied annotation.
	Notes string
}

// newOrigin returns a snapshot of where backup was run.  That
// snapshot is a new backup Origin value, for use in a backup's
// metadata.  Every value except for the machine name is populated
// either from juju state or some other implicit mechanism.
func NewMetadataState(st *state.State, machine string) (*Metadata, error) {
	// hostname could be derived from the environment...
	hostname, err := os.Hostname()
	if err != nil {
		// If os.Hostname() is not working, something is woefully wrong.
		// Run for the hills.
		return nil, errors.Annotate(err, "could not get hostname (system unstable?)")
	}

	meta := NewMetadata()

	meta.Origin.Environment = st.EnvironTag().Id()
	meta.Origin.Machine = machine
	meta.Origin.Hostname = hostname

	return meta, nil
}

// NewMetadata returns a new Metadata for a state backup archive.  The
// current date/time is used for the timestamp and the default checksum
// format is used.  ID is not set.  That is left up to the persistence
// layer.  Stored is set as false.  "notes" may be empty, but
// everything else should be provided.
func NewMetadata() *Metadata {
	return &Metadata{
		FileMetadata: filestorage.NewMetadata(),
		Origin:       newOrigin(),
		Started:      time.Now().UTC(),
	}
}

// Copy returns a deep copy of the metadata.
func (m *Metadata) Copy() filestorage.Document {
	fileMeta := m.FileMetadata.Copy().(*filestorage.FileMetadata)
	copied := *m
	copied.FileMetadata = fileMeta
	return &copied
}

type flatMetadata struct {
	ID string

	// file storage

	Checksum       string
	ChecksumFormat string
	Size           int64
	Stored         time.Time

	// backup

	Started     time.Time
	Finished    time.Time
	Notes       string
	Environment string
	Machine     string
	Hostname    string
	Version     version.Number
}

// TODO(ericsnow) Move AsJSONBuffer to filestorage.Metadata.

// AsJSONBuffer returns a bytes.Buffer containing the JSON-ified metadata.
func (m *Metadata) AsJSONBuffer() (io.Reader, error) {
	flat := flatMetadata{
		ID: m.ID(),

		Checksum:       m.Checksum(),
		ChecksumFormat: m.ChecksumFormat(),
		Size:           m.Size(),

		Started:     m.Started,
		Notes:       m.Notes,
		Environment: m.Origin.Environment,
		Machine:     m.Origin.Machine,
		Hostname:    m.Origin.Hostname,
		Version:     m.Origin.Version,
	}

	stored := m.Stored()
	if stored != nil {
		flat.Stored = *stored
	}

	if m.Finished != nil {
		flat.Finished = *m.Finished
	}

	var outfile bytes.Buffer
	if err := json.NewEncoder(&outfile).Encode(flat); err != nil {
		return nil, errors.Trace(err)
	}
	return &outfile, nil
}
