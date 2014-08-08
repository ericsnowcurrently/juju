// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"compress/gzip"
	"crypto/sha1"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/juju/errors"
	"github.com/juju/utils/filestorage"
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

type newBackup struct {
	tempdir  string
	bkpdir   string
	dumpdir  string
	tempfile string
	archive  io.WriteCloser
	rootfile io.WriteCloser
}

func (b *newBackup) prepare() error {
	tempDir, err := ioutil.TempDir("", "jujuBackup")
	if err != nil {
		return errors.Annotate(err, "error creating root temp directory")
	}
	bkpDir := filepath.Join(tempDir, "juju-backup")
	dumpDir := filepath.Join(bkpDir, "dump")
	tempfile := filepath.Join(tempDir, "backup.tar.gz")
	// We go with user-only permissions on principle; the directories
	// are short-lived so in practice it shouldn't matter much.
	err = os.MkdirAll(dumpDir, os.FileMode(0755))
	if err != nil {
		return errors.Annotate(err, "error creating backup temp directory")
	}
	// We create these files here in order to fail as early as possible.
	archive, err := os.Create(tempfile)
	if err != nil {
		return errors.Annotate(err, "error creating backup archive file")
	}
	rootfile, err := os.Create(filepath.Join(bkpDir, "root.tar"))
	if err != nil {
		return errors.Annotate(err, "error creating root archive file")
	}

	b.tempdir = tempDir
	b.bkpdir = bkpDir
	b.dumpdir = dumpDir
	b.tempfile = tempfile
	b.rootfile = rootfile
	b.archive = archive
	return nil
}

func (b *newBackup) cleanUp() error {
	if b.tempdir != "" {
		err := os.RemoveAll(b.tempdir)
		if err != nil {
			return errors.Annotate(err, "error deleting temp directory")
		}
		b.tempdir = ""
	}
	if b.rootfile != nil {
		err := b.rootfile.Close()
		if err != nil {
			return errors.Annotate(err, "error closing root file")
		}
		b.rootfile = nil
	}
	if b.archive != nil {
		err := b.archive.Close()
		if err != nil {
			return errors.Annotate(err, "error closing archive file")
		}
		b.archive = nil
	}
	return nil
}

func (b *newBackup) buildFilesBundle(config config.BackupsConfig) error {
	filenames, err := getFilesToBackup(config)
	if err != nil {
		return errors.Trace(err)
	}

	stripPrefix := string(os.PathSeparator)
	_, err = tar.TarFiles(filenames, b.rootfile, stripPrefix)
	if err != nil {
		return errors.Annotate(err, "error bundling state files")
	}

	// We explicitly close the file here so that it is completely
	// flushed to disk when we build the final archive.
	err = b.rootfile.Close()
	if err != nil {
		return errors.Annotate(err, "error closing root file")
	}
	b.rootfile = nil

	return nil
}

func (b *newBackup) dumpDB(config config.BackupsConfig) error {
	cmd, args, err := getDumpCmd(config, b.dumpdir)
	if err != nil {
		return errors.Trace(err)
	}

	err = runCommand(cmd, args...)
	if err != nil {
		return errors.Annotate(err, "error dumping database")
	}

	return nil
}

func (b *newBackup) buildArchive() (string, error) {
	// Build the tarball, writing out to both the archive file and a
	// SHA1 hash.  The hash will correspond to the gzipped file rather
	// than to the uncompressed contents of the tarball.  This is so
	// that users can compare the published checksum against the
	// checksum of the file without having to decompress it first.
	hashed := hash.NewHashingWriter(b.archive, sha1.New())
	err := func() error {
		tarball := gzip.NewWriter(hashed)
		// XXX Don't let errors pass silently here.
		defer tarball.Close()
		// We add a trailing slash (or whatever) to root so that everything
		// in the path up to and including that slash is stripped off when
		// each file is added to the tar file.
		stripPrefix := b.tempdir + string(os.PathSeparator)
		_, err := tar.TarFiles([]string{b.bkpdir}, tarball, stripPrefix)
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
	err = b.archive.Close()
	if err != nil {
		return "", errors.Annotate(err, "error closing archive file")
	}
	b.archive = nil

	// Gzip writers may buffer what they're writing so we must call
	// Close() on the writer *before* getting the checksum from the
	// hasher.
	checksum := hashed.Base64Sum()
	return checksum, nil
}

func (b *newBackup) run(
	config config.BackupsConfig,
) (string, io.ReadCloser, int64, error) {
	var err error

	// Bundle up the state-related files.
	logger.Infof("dumping state-related files")
	if err := b.buildFilesBundle(config); err != nil {
		return "", nil, 0, errors.Trace(err)
	}

	// Dump the database.
	logger.Infof("dumping database")
	if err := b.dumpDB(config); err != nil {
		return "", nil, 0, errors.Trace(err)
	}

	// Build the final tarball.
	logger.Infof("building archive file (%s)", b.tempfile)
	checksum, err := b.buildArchive()
	if err != nil {
		return "", nil, 0, errors.Trace(err)
	}

	// Re-open the archive for reading.
	archive, err := os.Open(b.tempfile)
	if err != nil {
		return "", nil, 0, errors.Annotate(err, "error opening temp archive")
	}
	stat, err := archive.Stat()
	if err != nil {
		return "", nil, 0, errors.Annotate(err, "error reading file info")
	}
	size := stat.Size()

	return checksum, archive, size, nil
}

// create creates a new backup archive (.tar.gz).  The backup contents
// look like this:
//   juju-backup/dump/ - the files generated from dumping the database
//   juju-backup/root.tar - contains all the files needed by juju
// Between the two, this is all that is necessary to later restore the
// juju agent on another machine.
func create(
	conf config.BackupsConfig, meta filestorage.Metadata,
) (io.ReadCloser, error) {
	creator := &newBackup{}
	err := creator.prepare()
	if err != nil {
		return nil, errors.Annotate(err, "error preparing for backup")
	}
	defer creator.cleanUp()

	// Create the archive.
	checksum, archive, size, err := creator.run(conf)
	if err != nil {
		return nil, errors.Annotate(err, "backup failed")
	}

	// Update the metadata.
	meta.SetFile(size, checksum, "SHA-1")

	return archive, nil
}
