// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"fmt"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/names"
	"labix.org/v2/mgo/txn"
)

type actionDoc struct {
	// Id is the key for this document.  Action.Id() has a specfic form
	// to facilitate filtering the actions collection for a given unit,
	// or in the future a given service.
	// The format of the Action.Id() will be:
	//   <unit globalKey> + actionMarker + <generated state sequence>
	Id string `bson:"_id"`

	// Name identifies the action; it should match an action defined by
	// the unit's charm.
	Name string

	// Payload holds the action's parameters, if any; it should validate
	// against the schema defined by the named action in the unit's charm
	Payload map[string]interface{}
}

// Action represents an instruction to do some "action" and is expected
// to match an action definition in a charm.
type Action struct {
	st  *State
	doc actionDoc
}

// newAction builds an Action from the supplied state and actionDoc
func newAction(st *State, adoc actionDoc) *Action {
	return &Action{
		st:  st,
		doc: adoc,
	}
}

// actionPrefix returns a suitable prefix for an action given the
// prefix of the Name() of a containing item
func actionPrefix(prefix string) string {
	return prefix + names.ActionMarker
}

// newActionId generates a new unique key from another entity name as
// a prefix, and a generated unique number
func newActionId(st *State, name string) (string, error) {
	prefix := actionPrefix(name)
	suffix, err := st.sequence(prefix)
	if err != nil {
		return "", errors.Errorf("cannot assign new sequence for prefix '%s': %v", prefix, err)
	}
	return fmt.Sprintf("%s%d", prefix, suffix), nil
}

// getActionIdPrefix returns the prefix for the given action id.
// Useful when finding a prefix to filter on.
func getActionIdPrefix(actionId string) string {
	return strings.Split(actionId, names.ActionMarker)[0]
}

// Id returns the id of the Action
func (a *Action) Id() string {
	return a.doc.Id
}

// Name returns the name of the Action
func (a *Action) Name() string {
	return a.doc.Name
}

// Tag implements the Entity interface and returns an ActionTag representation.
func (a *Action) Tag() names.Tag {
	return names.NewActionTag(a.Id())
}

// Payload will contain a structure representing arguments or parameters to
// an action, and is expected to be validated by the Unit using the Charm
// definition of the Action
func (a *Action) Payload() map[string]interface{} {
	return a.doc.Payload
}

// Complete removes action from the pending queue and creates an ActionResult
// to capture the output and end state of the action.
func (a *Action) Complete(output string) error {
	return a.removeAndLog(ActionCompleted, output)
}

// Fail removes an Action from the queue, and creates an ActionResult that
// will capture the reason for the failure.
func (a *Action) Fail(reason string) error {
	return a.removeAndLog(ActionFailed, reason)
}

// removeAndLog takes the action off of the pending queue, and creates an
// actionresult to capture the outcome of the action.
func (a *Action) removeAndLog(finalStatus ActionStatus, output string) error {
	result, err := newActionResultDoc(a, finalStatus, output)
	if err != nil {
		return err
	}
	return a.st.runTransaction([]txn.Op{
		addActionResultOp(a.st, result),
		{
			C:      a.st.actions.Name,
			Id:     a.doc.Id,
			Remove: true,
		},
	})
}
