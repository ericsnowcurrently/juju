// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"github.com/juju/juju/rpc/simple"
)

var APIBinding = simple.APIBinding{"backup", "POST", CompressionType}
