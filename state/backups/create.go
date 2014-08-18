// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"io"

	"github.com/juju/errors"
	"github.com/juju/utils/filestorage"

	"github.com/juju/juju/state/backups/config"
	build "github.com/juju/juju/state/backups/create"
)

var (
	newBuilder = func(config config.BackupsConfig) (build.Builder, error) {
		return build.NewBuilder(config)
	}
)

// create creates a new backup archive (.tar.gz).  The backup contents
// look like this:
//   juju-backup/dump/ - the files generated from dumping the database
//   juju-backup/root.tar - contains all the files needed by juju
// Between the two, this is all that is necessary to later restore the
// juju agent on another machine.
func create(
	config config.BackupsConfig, meta filestorage.Metadata,
) (io.ReadCloser, error) {
	builder, err := newBuilder(config)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer builder.Close()

	// Create the archive.
	result, err := builder.Build()
	if err != nil {
		return nil, errors.Annotate(err, "backup failed")
	}

	// Update the metadata.
	meta.SetFile(result.Size, result.Checksum, "SHA-1")

	return result.ArchiveFile, nil
}
