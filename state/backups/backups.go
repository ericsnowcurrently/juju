// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

/*
Package backups contains all the stand-alone backup-related
functionality for juju state. That functionality is encapsulated by
the backups.Backups type. The package also exposes a few key helpers
and components.

Backups are not a part of juju state nor of normal state operations.
However, they certainly are tightly coupled with state (the very
subject of backups). This puts backups in an odd position, particularly
with regard to the storage of backup metadata and archives.

As noted above backups are about state but not a part of state. So
exposing backup-related methods on State would imply the wrong thing.
Thus most of the functionality here is defined at a high level without
relation to state. A few low-level parts or helpers are exposed as
functions to which you pass a state value. Those are kept to a minimum.

Note that state (and juju as a whole) currently does not have a
persistence layer abstraction to facilitate separating different
persistence needs and implementations. As a consequence, state's
data, whether about how an environment should look or about existing
resources within an environment, is dumped essentially straight into
State's mongo connection. The code in the state package does not
make any distinction between the two (nor does the package clearly
distinguish between state-related abstractions and state-related
data).

Backups add yet another category, merely taking advantage of State's
mongo for storage. In the interest of making the distinction clear,
among other reasons, backups uses its own database under state's mongo
connection.
*/
package backups

import (
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/utils/filestorage"
)

const (
	// FilenamePrefix is the prefix used for backup archive files.
	FilenamePrefix = "juju-backup-"

	// FilenameTemplate is used with time.Time.Format to generate a filename.
	FilenameTemplate = FilenamePrefix + "20060102-150405.tar.gz"
)

var logger = loggo.GetLogger("juju.state.backups")

var (
	finishMeta = func(meta *Metadata, result *CreateResult) error {
		return meta.MarkComplete(result.Size, result.Checksum)
	}
	storeArchive = StoreArchive
)

// CreateResult holds the result of creating a new backup archive.
type CreateResult struct {
	// Archive is the new backup archive.
	Archive io.ReadCloser
	// Size is the size of the backup archive.
	Size int64
	// Checksum is the checksum of the archive.
	Checksum string
}

// Creator is used to create a new backup archive.
type Creator interface {
	// Create creates a new backup archive.
	Create(meta *Metadata) (*CreateResult, error)
}

type Restorer interface {
	Restore(archive io.Reader) error
}

// StoreArchive sends the backup archive and its metadata to storage.
// It also sets the metadata's ID and Stored values.
func StoreArchive(stor filestorage.FileStorage, meta *Metadata, file io.Reader) error {
	id, err := stor.Add(meta, file)
	if err != nil {
		return errors.Trace(err)
	}
	meta.SetID(id)
	stored, err := stor.Metadata(id)
	if err != nil {
		return errors.Trace(err)
	}
	meta.SetStored(stored.Stored())
	return nil
}

// Backups is an abstraction around all juju backup-related functionality.
type Backups interface {

	// Create creates and stores a new juju backup archive. It updates
	// the provided metadata.
	Create(meta *Metadata, creator Creator) error

	// Get returns the metadata and archive file associated with the ID.
	Get(id string) (*Metadata, io.ReadCloser, error)

	// List returns the metadata for all stored backups.
	List() ([]*Metadata, error)

	// Remove deletes the backup from storage.
	Remove(id string) error

	// Restore restores juju state from a stored archive.
	Restore(id string, restorer Restorer) error
}

type backups struct {
	storage filestorage.FileStorage
}

// NewBackups creates a new Backups value using the FileStorage provided.
func NewBackups(stor filestorage.FileStorage) Backups {
	b := backups{
		storage: stor,
	}
	return &b
}

// Create creates and stores a new juju backup archive and updates the
// provided metadata.
func (b *backups) Create(meta *Metadata, creator Creator) error {
	meta.Started = time.Now().UTC()

	result, err := creator.Create(meta)
	if err != nil {
		return errors.Trace(err)
	}
	defer result.Archive.Close()

	// Finalize the metadata.
	err = finishMeta(meta, result)
	if err != nil {
		return errors.Annotate(err, "while updating metadata")
	}

	// Store the archive.
	err = storeArchive(b.storage, meta, result.Archive)
	if err != nil {
		return errors.Annotate(err, "while storing backup archive")
	}

	return nil
}

// Get pulls the associated metadata and archive file from environment storage.
func (b *backups) Get(id string) (*Metadata, io.ReadCloser, error) {
	if strings.HasPrefix(id, uploadedPrefix) {
		archiveFile, err := openUploaded(id)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		// TODO(ericsnow) Extract the metadata from the file.
		return nil, archiveFile, nil
	}

	rawmeta, archiveFile, err := b.storage.Get(id)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	meta, ok := rawmeta.(*Metadata)
	if !ok {
		return nil, nil, errors.New("did not get a backups.Metadata value from storage")
	}

	return meta, archiveFile, nil
}

// List returns the metadata for all stored backups.
func (b *backups) List() ([]*Metadata, error) {
	metaList, err := b.storage.List()
	if err != nil {
		return nil, errors.Trace(err)
	}
	result := make([]*Metadata, len(metaList))
	for i, meta := range metaList {
		m, ok := meta.(*Metadata)
		if !ok {
			msg := "expected backups.Metadata value from storage for %q, got %T"
			return nil, errors.Errorf(msg, meta.ID(), meta)
		}
		result[i] = m
	}
	return result, nil
}

// Remove deletes the backup from storage.
func (b *backups) Remove(id string) error {
	return errors.Trace(b.storage.Remove(id))
}

// Restore restores juju state from a stored archive.
func (b *backups) Restore(id string, restorer Restorer) error {
	_, archive, err := b.storage.Get(id)
	if err != nil {
		return errors.Trace(err)
	}
	defer archive.Close()

	err = restorer.Restore(archive)
	return errors.Trace(err)
}

// The following code is temporary and in support of the initial
// restore patch.  Once we have an HTTP-based upload this code will
// be removed.

const (
	uploadedPrefix = "file://"
	sshUsername    = "ubuntu"
)

type sendFunc func(host, filename string, archive io.Reader) error

// SimpleUpload sends the backup archive to the server where it is saved
// in the home directory of the SSH user.  The returned ID may be used
// to locate the file on the server.
func SimpleUpload(publicAddress string, archive io.Reader, send sendFunc) (string, error) {
	filename := time.Now().UTC().Format(FilenameTemplate)
	host := sshUsername + "@" + publicAddress
	err := send(host, filename, archive)
	return uploadedPrefix + filename, errors.Trace(err)
}

func resolveUploaded(id string) (string, error) {
	filename := strings.TrimPrefix(id, uploadedPrefix)
	filename = filepath.FromSlash(filename)
	if !strings.HasPrefix(filepath.Base(filename), FilenamePrefix) {
		return "", errors.Errorf("invalid ID for uploaded file: %q", id)
	}
	if filepath.IsAbs(filename) {
		return "", errors.Errorf("expected relative path in ID, got %q", id)
	}

	sshUser, err := user.Lookup(sshUsername)
	if err != nil {
		return "", errors.Trace(err)
	}
	filename = filepath.Join(sshUser.HomeDir, filename)
	return filename, nil
}

func openUploaded(id string) (io.ReadCloser, error) {
	filename, err := resolveUploaded(id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	archive, err := os.Open(filename)
	return archive, errors.Trace(err)
}
