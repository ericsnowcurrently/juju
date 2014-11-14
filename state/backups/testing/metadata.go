// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"time"

	"github.com/juju/juju/state/backups"
)

const (
	envID = "49db53ac-a42f-4ab2-86e1-0c6fa0fec762"
)

// NewMetadata returns a Metadata to use for testing.
func NewMetadata() *backups.Metadata {
	timestamp := "20140924-010319"
	id := envID + "." + timestamp
	notes := ""
	meta := NewMetadataStarted(id, notes)

	FinishMetadata(meta)
	meta.SetStored(nil)
	return meta
}

// NewMetadataStarted returns a Metadata to use for testing.
func NewMetadataStarted(id, notes string) *backups.Metadata {
	meta := backups.NewMetadata()
	meta.Origin.Environment = envID
	meta.Origin.Machine = "0"
	meta.Origin.Hostname = "main-host"
	meta.Started = time.Now().UTC()
	meta.SetID(id)
	return meta
}

// FinishMetadata finishes a metadata with test values.
func FinishMetadata(meta *backups.Metadata) {
	var size int64 = 10
	checksum := "787b8915389d921fa23fb40e16ae81ea979758bf"
	backups.UpdateMetadata(meta, size, checksum)
	finished := meta.Started.Add(time.Minute)
	meta.Finished = &finished
}

// UpdateNotes derives a new Metadata with new notes.
func UpdateNotes(meta *backups.Metadata, notes string) *backups.Metadata {
	copied := meta.Copy().(*backups.Metadata)
	copied.SetID(meta.ID())
	copied.Notes = notes
	return copied
}
