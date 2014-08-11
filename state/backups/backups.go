// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// backups contains all the stand-alone backup-related functionality for
// juju state.
package backups

import (
	"github.com/juju/loggo"
	"github.com/juju/utils/filestorage"

	"github.com/juju/juju/state/backups/config"
)

var logger = loggo.GetLogger("juju.state.backup")

// Backups is an abstraction around all juju backup-related functionality.
type Backups interface {
}

type backups struct {
	config  config.BackupsConfig
	storage filestorage.FileStorage
}

// NewBackups returns a new Backups value using the provided DB info and
// file storage.
func NewBackups(
	config config.BackupsConfig, stor filestorage.FileStorage,
) Backups {
	b := backups{
		config:  config,
		storage: stor,
	}
	return &b
}
