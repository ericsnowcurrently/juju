// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package params

import (
	"time"

	"github.com/juju/juju/state/backup"
	"github.com/juju/juju/version"
)

// BackupCreateArgs holds the args for the API Create method.
type BackupCreateArgs struct {
	Notes string
}

// BackupInfoArgs holds the args for the API Info method.
type BackupInfoArgs struct {
	ID string
}

// BackupListArgs holds the args for the API List method.
type BackupListArgs struct {
}

// BackupDownloadArgs holds the args for the API Download method.
type BackupDownloadArgs struct {
	ID string
}

// BackupRemoveArgs holds the args for the API Remove method.
type BackupRemoveArgs struct {
	ID string
}

// BackupListResult holds the list of all stored backups.
type BackupListResult struct {
	List []BackupMetadataResult
}

// BackupMetadataResult holds the metadata for a backup as returned by
// an API backups method (such as Create).
type BackupMetadataResult struct {
	ID string

	Checksum       string
	ChecksumFormat string
	Size           int64
	Stored         time.Time // May be zero...

	Started     time.Time
	Finished    time.Time // May be zero...
	Notes       string
	Environment string
	Machine     string
	Hostname    string
	Version     version.Number
}

// UpdateFromMetadata updates the result with the information in the
// metadata value.
func (r *BackupMetadataResult) UpdateFromMetadata(meta *backup.Metadata) {
	r.ID = meta.ID()

	r.Checksum = meta.Checksum()
	r.ChecksumFormat = meta.ChecksumFormat()
	r.Size = meta.Size()
	if meta.Stored() != nil {
		r.Stored = *(meta.Stored())
	}

	r.Started = meta.Started
	if meta.Finished != nil {
		r.Finished = *meta.Finished
	}
	r.Notes = meta.Notes

	r.Environment = meta.Origin.Environment
	r.Machine = meta.Origin.Machine
	r.Hostname = meta.Origin.Hostname
	r.Version = meta.Origin.Version
}
