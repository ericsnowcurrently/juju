// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"io"
	"time"

	"github.com/juju/errors"
	"github.com/juju/testing"
	"github.com/juju/utils/filestorage"

	"github.com/juju/juju/state"
)

var (
	FinishMeta         = &finishMeta
	StoreArchiveRef    = &storeArchive
	ReplaceableFolders = &replaceableFolders
)

var _ filestorage.DocStorage = (*backupsDocStorage)(nil)
var _ filestorage.RawFileStorage = (*backupBlobStorage)(nil)

func getBackupDBWrapper(st *state.State) *storageDBWrapper {
	envUUID := st.EnvironTag().Id()
	db := st.MongoSession().DB(storageDBName)
	return newStorageDBWrapper(db, storageMetaName, envUUID)
}

// NewBackupID creates a new backup ID based on the metadata.
func NewBackupID(meta *Metadata) string {
	doc := newStorageMetaDoc(meta)
	return newStorageID(&doc)
}

// GetBackupMetadata returns the metadata retrieved from storage.
func GetBackupMetadata(st *state.State, id string) (*Metadata, error) {
	db := getBackupDBWrapper(st)
	defer db.Close()
	doc, err := getStorageMetadata(db, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return docAsMetadata(doc), nil
}

// AddBackupMetadata adds the metadata to storage.
func AddBackupMetadata(st *state.State, meta *Metadata) (string, error) {
	db := getBackupDBWrapper(st)
	defer db.Close()
	doc := newStorageMetaDoc(meta)
	return addStorageMetadata(db, &doc)
}

// AddBackupMetadataID adds the metadata to storage, using the given
// backup ID.
func AddBackupMetadataID(st *state.State, meta *Metadata, id string) error {
	restore := testing.PatchValue(&newStorageID, func(*storageMetaDoc) string {
		return id
	})
	defer restore()

	db := getBackupDBWrapper(st)
	defer db.Close()
	doc := newStorageMetaDoc(meta)
	_, err := addStorageMetadata(db, &doc)
	return errors.Trace(err)
}

// SetBackupStoredTime stores the time of when the identified backup archive
// file was stored.
func SetBackupStoredTime(st *state.State, id string, stored time.Time) error {
	db := getBackupDBWrapper(st)
	defer db.Close()
	return setStorageStoredTime(db, id, stored)
}

// NewTestMetaFinisher builds a new replacement for finishMetadata with
// the given failure.
func NewTestMetaFinisher(failure string) func(*Metadata, *CreateResult) error {
	return func(*Metadata, *CreateResult) error {
		if failure == "" {
			return nil
		}
		return errors.New(failure)
	}
}

// NewTestArchiveStorer builds a new replacement for StoreArchive with
// the given failure.
func NewTestArchiveStorer(failure string) func(filestorage.FileStorage, *Metadata, io.Reader) error {
	return func(filestorage.FileStorage, *Metadata, io.Reader) error {
		if failure == "" {
			return nil
		}
		return errors.New(failure)
	}
}
