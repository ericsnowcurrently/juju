// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"io"
	"os"
	"path"
	"time"

	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/utils/filestorage"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/txn"

	"github.com/juju/juju/environs/storage"
	"github.com/juju/juju/state/backup"
	"github.com/juju/juju/version"
)

/*
Backups are not a part of juju state nor of normal state operations.
However, they certainly are tightly coupled with state (the very
subject of backups).  This puts backups in an odd position,
particularly with regard to the storage of backup metadata and
archives.  As a result, here are a couple concerns worth mentioning.

First, as noted above backup is about state but not a part of state.
So exposing backup-related methods on State would imply the wrong
thing.  Thus the backup functionality here in the state package (not
state/backup) is exposed as functions to which you pass a state
object.

Second, backup creates an archive file containing a dump of state's
mongo DB.  Storing backup metadata/archives in mongo is thus a
somewhat circular proposition that has the potential to cause
problems.  That may need further attention.

Note that state (and juju as a whole) currently does not have a
persistence layer abstraction to facilitate separating different
persistence needs and implementations.  As a consequence, state's
data, whether about how an environment should look or about existing
resources within an environment, is dumped essentially straight into
State's mongo connection.  The code in the state package does not
make any distinction between the two (nor does the package clearly
distinguish between state-related abstractions and state-related
data).

Backup adds yet another category, merely taking advantage of
State's DB.  In the interest of making the distinction clear, the
code that directly interacts with State (and its DB) lives in this
file.  As mentioned previously, the functionality here is exposed
through functions that take State, rather than as methods on State.
Furthermore, the bulk of the backup code, which does not need direct
interaction with State, lives in the state/backup package.
*/

// backupMetadataDoc is a mirror of backup.Metadata, used just for DB storage.
type backupMetadataDoc struct {
	ID             string `bson:"_id"`
	Started        int64  `bson:"started,minsize"`
	Finished       int64  `bson:"finished,minsize"`
	Checksum       string `bson:"checksum"`
	ChecksumFormat string `bson:"checksumformat"`
	Size           int64  `bson:"size,minsize"`
	Stored         bool   `bson:"stored"`
	Notes          string `bson:"notes,omitempty"`

	// origin
	Environment string         `bson:"environment"`
	Machine     string         `bson:"machine"`
	Hostname    string         `bson:"hostname"`
	Version     version.Number `bson:"version"`
}

// asMetadata returns a new backup.Metadata based on the backupMetadataDoc.
func (doc *backupMetadataDoc) asMetadata() *backup.Metadata {
	origin := backup.NewOrigin(
		doc.Environment,
		doc.Machine,
		doc.Hostname,
		doc.Version,
	)
	var finished *time.Time
	if doc.Finished != 0 {
		val := time.Unix(doc.Finished, 0).UTC()
		finished = &val
	}
	metadata := backup.NewMetadataFull(
		doc.Size,
		doc.Checksum,
		*origin,
		doc.Notes,
		doc.ChecksumFormat,
		time.Unix(doc.Started, 0).UTC(),
		finished,
	)
	metadata.SetID(doc.ID)
	if doc.Stored {
		metadata.SetStored()
	}
	return metadata
}

// updateFromMetadata copies the corresponding data from the backup.Metadata
// into the backupMetadataDoc.
func (doc *backupMetadataDoc) updateFromMetadata(metadata *backup.Metadata) {
	finished := metadata.Finished()
	// Ignore metadata.ID.
	doc.Started = metadata.Started().Unix()
	if finished != nil {
		doc.Finished = finished.Unix()
	}
	doc.Checksum = metadata.Checksum()
	doc.ChecksumFormat = metadata.ChecksumFormat()
	doc.Size = metadata.Size()
	doc.Stored = metadata.Stored()
	doc.Notes = metadata.Notes()

	origin := metadata.Origin()
	doc.Environment = origin.Environment()
	doc.Machine = origin.Machine()
	doc.Hostname = origin.Hostname()
	doc.Version = origin.Version()
}

//---------------------------
// DB operations

// getBackupMetadata returns the backup metadata associated with "id".
// If "id" does not match any stored records, an error satisfying
// juju/errors.IsNotFound() is returned.
func getBackupMetadata(st *State, id string) (*backup.Metadata, error) {
	collection, closer := st.getCollection(backupsC)
	defer closer()

	var doc backupMetadataDoc
	// There can only be one!
	err := collection.FindId(id).One(&doc)
	if err == mgo.ErrNotFound {
		return nil, errors.NotFoundf(id)
	} else if err != nil {
		return nil, errors.Annotate(err, "error getting backup metadata")
	}

	return doc.asMetadata(), nil
}

// addBackupMetadata stores metadata for a backup where it can be
// accessed later.  It returns a new ID that is associated with the
// backup.  If the provided metadata already has an ID set, it is
// ignored.
func addBackupMetadata(st *State, metadata *backup.Metadata) (string, error) {
	// We use our own mongo _id value since the auto-generated one from
	// mongo may contain sensitive data (see bson.ObjectID).
	id, err := utils.NewUUID()
	if err != nil {
		return "", errors.Annotate(err, "error generating new ID")
	}
	idStr := id.String()
	return idStr, addBackupMetadataID(st, metadata, idStr)
}

func addBackupMetadataID(st *State, metadata *backup.Metadata, id string) error {
	var doc backupMetadataDoc
	doc.updateFromMetadata(metadata)
	doc.ID = id

	ops := []txn.Op{{
		C:      backupsC,
		Id:     doc.ID,
		Assert: txn.DocMissing,
		Insert: doc,
	}}
	if err := st.runTransaction(ops); err != nil {
		if err == txn.ErrAborted {
			return errors.AlreadyExistsf(doc.ID)
		}
		return errors.Annotate(err, "error running transaction")
	}

	return nil
}

// setBackupStored updates the backup metadata associated with "id"
// to indicate that a backup archive has been stored.  If "id" does
// not match any stored records, an error satisfying
// juju/errors.IsNotFound() is returned.
func setBackupStored(st *State, id string) error {
	ops := []txn.Op{{
		C:      backupsC,
		Id:     id,
		Assert: txn.DocExists,
		Update: bson.D{{"$set", bson.D{
			{"stored", true},
		}}},
	}}
	if err := st.runTransaction(ops); err != nil {
		if err == txn.ErrAborted {
			return errors.NotFoundf(id)
		}
		return errors.Annotate(err, "error running transaction")
	}
	return nil
}

//---------------------------
// metadata storage

func newBackupOrigin(st *State, machine string) *backup.Origin {
	hostname, err := os.Hostname()
	if err != nil {
		// Ignore the error.
		hostname = ""
	}
	return backup.NewOrigin(
		st.EnvironTag().Id(),
		machine,
		hostname,
		version.Current.Number,
	)
}

type backupMetadataStorage struct {
	state *State
}

func newBackupMetadataStorage(st *State) filestorage.MetadataStorage {
	stor := backupMetadataStorage{
		state: st,
	}
	return &stor
}

func (s *backupMetadataStorage) AddDoc(doc interface{}) (string, error) {
	metadata, ok := doc.(backup.Metadata)
	if !ok {
		return "", errors.Errorf("doc must be of type state.backup.Metadata")
	}
	return addBackupMetadata(s.state, &metadata)
}

func (s *backupMetadataStorage) Doc(id string) (interface{}, error) {
	return s.Metadata(id)
}

func (s *backupMetadataStorage) Metadata(id string) (filestorage.Metadata, error) {
	metadata, err := getBackupMetadata(s.state, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return metadata, nil
}

func (s *backupMetadataStorage) ListDocs() ([]interface{}, error) {
	metas, err := s.ListMetadata()
	if err != nil {
		return nil, errors.Trace(err)
	}
	docs := []interface{}{}
	for _, meta := range metas {
		docs = append(docs, meta)
	}
	return docs, nil
}

func (s *backupMetadataStorage) ListMetadata() ([]filestorage.Metadata, error) {
	return nil, errors.New("not implemented yet")
}

func (s *backupMetadataStorage) RemoveDoc(id string) error {
	return errors.New("not implemented yet")
}

func (s *backupMetadataStorage) New() filestorage.Metadata {
	origin := newBackupOrigin(s.state, "")
	return backup.NewMetadata(0, "", *origin, "")
}

func (s *backupMetadataStorage) SetStored(meta filestorage.Metadata) error {
	err := setBackupStored(s.state, meta.ID())
	if err != nil {
		return errors.Trace(err)
	}
	meta.SetStored()
	return nil
}

//---------------------------
// raw file storage

const backupStorageRoot = "/"

type envFileStorage struct {
	envStor storage.Storage
	root    string
}

func newBackupFileStorage(envStor storage.Storage, root string) filestorage.RawFileStorage {
	// Due to circular imports we cannot simply get the storage from
	// State using environs.GetStorage().
	stor := envFileStorage{
		envStor: envStor,
		root:    root,
	}
	return &stor
}

func (s *envFileStorage) path(id string) string {
	// Use of path.Join instead of filepath.Join is intentional - this
	// is an environment storage path not a filesystem path.
	return path.Join(s.root, id)
}

func (s *envFileStorage) File(id string) (io.ReadCloser, error) {
	return s.envStor.Get(s.path(id))
}

func (s *envFileStorage) AddFile(id string, file io.Reader, size int64) error {
	return s.envStor.Put(s.path(id), file, size)
}

func (s *envFileStorage) RemoveFile(id string) error {
	return s.envStor.Remove(s.path(id))
}

//---------------------------
// backup storage

func NewBackupStorage(st *State, envStor storage.Storage) filestorage.FileStorage {
	files := newBackupFileStorage(envStor, backupStorageRoot)
	docs := newBackupMetadataStorage(st)
	return filestorage.NewFileStorage(docs, files)
}
