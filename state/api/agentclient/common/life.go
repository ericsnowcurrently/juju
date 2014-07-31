// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common

import (
	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/juju/state/api/base"
	"github.com/juju/juju/state/api/params"
)

// Life requests the life cycle of the given entity from the given
// server-side API facade via the given caller.
func Life(caller base.Caller, facadeName string, tag names.Tag) (params.Life, error) {
	var result params.LifeResults
	args := params.Entities{
		Entities: []params.Entity{{Tag: tag.String()}},
	}
	if err := caller.Call(facadeName, "", "Life", args, &result); err != nil {
		return "", err
	}
	if len(result.Results) != 1 {
		return "", errors.Errorf("expected 1 result, got %d", len(result.Results))
	}
	if err := result.Results[0].Error; err != nil {
		return "", err
	}
	return result.Results[0].Life, nil
}
