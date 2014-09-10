// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"github.com/juju/errors"

	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/backups/db"
)

func (b *BackupsAPI) Create(args params.BackupsCreateArgs) (
	p params.BackupsMetadataResult, err error,
) {
	mgoInfo := b.st.MongoConnectionInfo()
	dbInfo := db.NewMongoConnInfo(mgoInfo)

	// XXX Get from state.
	machine := ""
	origin := state.NewBackupsOrigin(b.st, machine)

	meta, err := b.backups.Create(*dbInfo, *origin, args.Notes)
	if err != nil {
		return p, errors.Trace(err)
	}

	p.UpdateFromMetadata(meta)

	return p, nil
}
