// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package workspace

import (
	"fmt"

	"github.com/juju/loggo"
)

// XXX This should be in the errors repo.

type MultiError struct {
	Errors []error
}

func NewMultiError(errs []error) *MultiError {
	if len(errs) == 0 {
		return nil
	}
	return &MultiError{errs}
}

func (e MultiError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	} else {
		return fmt.Sprintf("%d errors", len(e.Errors))
	}
}

func (e MultiError) Log(logger loggo.Logger) {
	if len(e.Errors) != 1 {
		logger.Errorf(e.Error())
	}
	for _, err := range e.Errors {
		logger.Errorf(err.Error())
	}
}
