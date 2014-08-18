// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package create

import (
	"compress/gzip"
	"crypto/sha1"
	"io"
	"os"

	"github.com/juju/errors"
	"github.com/juju/utils/hash"
	"github.com/juju/utils/tar"

	"github.com/juju/juju/state/backups/config"
	coreutils "github.com/juju/juju/utils"
)

var (
	runCommand       = coreutils.RunCommand
	getFilesToBackup = func(config config.BackupsConfig) ([]string, error) {
		return config.FilesToBackUp()
	}
	getDumpCmd = func(
		config config.BackupsConfig, outdir string,
	) (string, []string, error) {
		return config.DBDump(outdir)
	}
)

type Builder interface {
	io.Closer
	Build() (*Result, error)
}

type builder struct {
	ws     *workspace
	config config.BackupsConfig
}

// NewBuilder returns a new builder based on the config.
func NewBuilder(config config.BackupsConfig) (*builder, error) {
	// Create the workspace.
	ws, err := newWorkspace()
	if err != nil {
		return nil, errors.Annotate(err, "error preparing workspace")
	}

	// Create the archive files.  We create them here in order to fail
	// as early as possible.
	err = ws.createArchiveFile()
	if err != nil {
		ws.cleanUp()
		return nil, errors.Trace(err)
	}
	err = ws.createBundleFile()
	if err != nil {
		ws.cleanUp()
		return nil, errors.Trace(err)
	}

	// Create the builder.
	b := builder{
		ws:     ws,
		config: config,
	}
	return &b, nil
}

func (b *builder) Close() error {
	if err := b.ws.cleanUp(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (b *builder) buildFilesBundle() error {
	filenames, err := getFilesToBackup(b.config)
	if err != nil {
		return errors.Trace(err)
	}

	stripPrefix := string(os.PathSeparator)
	_, err = tar.TarFiles(filenames, b.ws.bundleFile, stripPrefix)
	if err != nil {
		return errors.Annotate(err, "error bundling state files")
	}

	// We explicitly close the file here so that it is completely
	// flushed to disk when we build the final archive.
	err = b.ws.closeBundleFile()
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (b *builder) dumpDB() error {
	cmd, args, err := getDumpCmd(b.config, b.ws.archive.DBDumpDir())
	if err != nil {
		return errors.Trace(err)
	}

	err = runCommand(cmd, args...)
	if err != nil {
		return errors.Annotate(err, "error dumping database")
	}

	return nil
}

func (b *builder) buildArchive() (string, error) {
	// Build the tarball, writing out to both the archive file and a
	// SHA1 hash.  The hash will correspond to the gzipped file rather
	// than to the uncompressed contents of the tarball.  This is so
	// that users can compare the published checksum against the
	// checksum of the file without having to decompress it first.
	hashed := hash.NewHashingWriter(b.ws.archiveFile, sha1.New())
	// We use a func literal here to make the tarball lifetime clear.
	// Closing the gzip writer early is important (see checksum below).
	err := func() error {
		tarball := gzip.NewWriter(hashed)
		// XXX Don't let errors pass silently when we close the tarball.
		defer tarball.Close()
		// We add a trailing slash (or whatever) to root so that everything
		// in the path up to and including that slash is stripped off when
		// each file is added to the tar file.
		stripPrefix := b.ws.archive.RootDir() + string(os.PathSeparator)
		filenames := []string{b.ws.archive.ContentDir()}
		_, err := tar.TarFiles(filenames, tarball, stripPrefix)
		if err != nil {
			return errors.Annotate(err, "error bundling final archive")
		}
		return nil
	}()
	if err != nil {
		return "", err
	}

	// We explicitly close the file here so that it is completely
	// flushed to disk when we go to re-open it in the next step.
	err = b.ws.closeArchiveFile()
	if err != nil {
		return "", errors.Trace(err)
	}

	// Gzip writers may buffer what they're writing so we must call
	// Close() on the writer *before* getting the checksum from the
	// hasher.
	checksum := hashed.Base64Sum()
	return checksum, nil
}

func (b *builder) Build() (*Result, error) {
	var err error

	// Bundle up the state-related files.
	logger.Infof("dumping state-related files")
	if err := b.buildFilesBundle(); err != nil {
		return nil, errors.Trace(err)
	}

	// Dump the database.
	logger.Infof("dumping database")
	if err := b.dumpDB(); err != nil {
		return nil, errors.Trace(err)
	}

	// Build the final tarball.
	logger.Infof("building archive file (%s)", b.ws.archive.Filename())
	checksum, err := b.buildArchive()
	if err != nil {
		return nil, errors.Trace(err)
	}

	result, err := NewResult(b.ws.archive, checksum)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return result, nil
}
