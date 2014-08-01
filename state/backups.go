// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"errors"
	"io"
	"path"

	"labix.org/v2/mgo" // XXX wrong import

	"github.com/juju/juju/environs"
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

// backupInfoDoc is a mirror of backup.Info, used just for DB storage.
type backupInfoDoc struct {
	ID        string `bson:"_id"`
	Notes     string
	Timestamp time.Time
	CheckSum  string
	Size      int64
	Version   version.Number
	Status    backup.Status
}

func (doc *backupInfoDoc) asInfo() *backup.Info {
	info := backup.Info{
		ID:        doc.ID,
		Notes:     doc.Notes,
		Timestamp: &doc.Timestamp,
		CheckSum:  doc.CheckSum,
		Size:      doc.Size,
		Version:   &doc.Version,
		Status:    doc.Status,
	}
	return &info
}

func (doc *backupInfoDoc) updateFromInfo(info *backup.Info) {
	doc.ID = info.ID
	doc.Notes = info.Notes
	doc.Timestamp = *info.Timestamp
	doc.CheckSum = info.CheckSum
	doc.Size = info.Size
	doc.Version = *info.Version
	doc.Status = info.Status
}

//---------------------------
// DB operations

var (
	ErrBackupMetadataNotFound = errors.New("backup info not found")
	ErrBackupMetadataNotAdded = errors.New("backup info not added")
	ErrBackupStatusNotUpdated = errors.New("backup status not updated")
)

func nextBackupID(st *State, timestamp *time.Time) string {
	return backup.IDFromTimestamp(timestamp, &version.Current.Number)
}

func updateStatusOp(id string, status backup.Status) txn.Op {
	updateFields := bson.D{{"$set", bson.D{
		{"status", status},
	}}}
	return txn.Op{
		C:      backupsC,
		Id:     id,
		Assert: txn.DocExists,
		Update: updateFields,
	}
}

func setBackupStatus(st *State, id string, status backup.Status) error {
	ops := []txn.Op{
		updateStatusOp(id, status),
	}
	if err := st.runTransaction(ops); err != nil {
		return onAbort(err, ErrBackupStatusNotUpdated)
	}

	return nil
}

func getBackupMetadata(st *State, id string) (*backup.Info, error) {
	collection, closer := st.getCollection(backupsC)
	defer closer()

	var doc backupInfoDoc
	// There can only be one!
	err := s.coll.FindId(id).One(&doc)
	if err == mg.ErrNotFound {
		return nil, ErrBackupMetadataNotFound
	} else if err != nil {
		return nil, fmt.Errorf("error getting backup metadata: %v", err)
	}

	return doc.asInfo(), nil
}

func addBackupMetadata(st *State, info *info.BackupInfo) (string, error) {
	var doc backupInfoDoc
	doc.updateFromInfo(info)
	doc.Name = nextBackupID(st, nil)
	doc.Status = backup.StatusStoringInfo
	status = info.Status
	if status == backup.StatusNotSet || status == backup.StatusStoringInfo {
		status = backup.StatusInfoOnly
	}

	ops := []txn.Op{
		{
			C:      backupsC,
			Id:     info.ID,
			Assert: txn.DocMissing,
			Insert: doc,
		},
		updateStatusOp(doc.ID, status),
	}
	if err := st.runTransaction(ops); err != nil {
		return onAbort(err, ErrBackupMetadataNotAdded)
	}

	return doc.ID, nil
}

//---------------------------
// metadata storage

type backupMetadataStorage struct {
	state *State
}

func newBackupMetadataStorage(st *State) *backupMetadataStorage {
	stor := backupMetadataStorage{
		state: st,
	}
	return &stor
}

func (s *backupMetadataStorage) AddDoc(id string, doc interface{}) (string, error) {
	if id != "" {
		return "", fmt.Errorf("explicit IDs are not supported")
	}

	info, ok := doc.(backup.Info)
	if !ok {
		return fmt.Errorf("doc must be of type state.backup.Info")
	}
	return AddBackupMetadata(s.state, &info)
}

func (s *backupMetadataStorage) Doc(id string, doc interface{}) error {
	info, err := GetBackupMetadata(s.state, id)
	if err != nil {
		return err
	}
	*doc = *info
	return nil
}

//---------------------------
// file storage

type envFileStorage struct {
	envStor storage.Storage
	root    string
}

func newBackupFileStorage(es storage.Storage, root string) backup.FileStorage {
	stor := envFileStorage{
		envStor: es,
		root:    root,
	}
	return &stor
}

func (s *envFileStorage) path(id string) string {
	// Use of path.Join instead of filepath.Join is intentional - this
	// is an environment storage path not a filesystem path.
	return path.Join(s.root, id)
}

func (s *envFileStorage) AddFile(id string, file io.Reader, size int64) error {
	return s.envStor.Put(s.path(id), file, size)
}

func (s *envFileStorage) File(id string) (io.ReadCloser, error) {
	return s.envStor.Get(s.path(id))
}

func (s *envFileStorage) URL(id string) (string, error) {
	return s.envStor.URL(s.path(id))
}

//---------------------------
// backup storage

type backupStorage struct {
	backup.WrappingBackupStorage
}

func NewBackupStorage(st *State) (backup.BackupStorage, error) {
	envStor, err := environs.GetStorage(st)
	if err != nil {
		return nil, fmt.Errorf("error getting environment storage: %v", err)
	}
	docs := newbackupMetadataStorage(st)
	files := newBackupFileStorage(envStor, backup.StorageRoot)
	stor := backup.NewBackupStorage(docs, files)
	return &backupStorage{stor}, nil
}

func (s *backupStorage) Add(info *BackupInfo, archive io.Reader) error {
}

//---------------------------
// DB info

func NewBackupsDBInfo(st *State) *backup.DBConnInfo {
	mgoInfo := st.MongoConnectionInfo()

	dbinfo := backup.DBConnInfo{
		Hostname: mgoInfo.Addrs[0],
		Password: mgoInfo.Password,
	}
	// TODO(dfc) Backup should take a Tag
	if mgoInfo.Tag != nil {
		dbinfo.Username = mgoInfo.Tag.String()
	}
	return &dbinfo
}
