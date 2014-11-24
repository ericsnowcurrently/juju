// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package create

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/juju/errors"
	"github.com/juju/utils/filestorage"

	"github.com/juju/juju/state/backups"
)

var (
	Create = create

	TestGetFilesToBackUp = &getFilesToBackUp
	GetDBDumper          = &getDBDumper
	RunCreate            = &runCreate
	GetMongodumpPath     = &getMongodumpPath
	RunCommand           = &runCommand
)

// NewTestCreateArgs builds a new args value for create() calls.
func NewTestCreateArgs(filesToBackUp []string, db DBDumper, metar io.Reader) *createArgs {
	args := createArgs{
		filesToBackUp:  filesToBackUp,
		db:             db,
		metadataReader: metar,
	}
	return &args
}

// ExposeCreateResult extracts the values in a create() args value.
func ExposeCreateArgs(args *createArgs) ([]string, DBDumper) {
	return args.filesToBackUp, args.db
}

// NewTestCreate builds a new replacement for create() with the given result.
func NewTestCreate(result *backups.CreateResult) (*createArgs, func(*createArgs) (*backups.CreateResult, error)) {
	var received createArgs

	if result == nil {
		archiveFile := ioutil.NopCloser(bytes.NewBufferString("<archive>"))
		result = &backups.CreateResult{archiveFile, 10, "<checksum>"}
	}

	testCreate := func(args *createArgs) (*backups.CreateResult, error) {
		received = *args
		return result, nil
	}

	return &received, testCreate
}

// NewTestCreate builds a new replacement for create() with the given failure.
func NewTestCreateFailure(failure string) func(*createArgs) (*backups.CreateResult, error) {
	return func(*createArgs) (*backups.CreateResult, error) {
		return nil, errors.New(failure)
	}
}

// NewTestMetaFinisher builds a new replacement for finishMetadata with
// the given failure.
func NewTestMetaFinisher(failure string) func(*backups.Metadata, *backups.CreateResult) error {
	return func(*backups.Metadata, *backups.CreateResult) error {
		if failure == "" {
			return nil
		}
		return errors.New(failure)
	}
}

// NewTestArchiveStorer builds a new replacement for StoreArchive with
// the given failure.
func NewTestArchiveStorer(failure string) func(filestorage.FileStorage, *backups.Metadata, io.Reader) error {
	return func(filestorage.FileStorage, *backups.Metadata, io.Reader) error {
		if failure == "" {
			return nil
		}
		return errors.New(failure)
	}
}
