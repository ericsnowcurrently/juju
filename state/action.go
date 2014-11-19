// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"time"

	"github.com/juju/names"
	"github.com/juju/utils"
	"gopkg.in/mgo.v2/txn"
)

// ActionReceiver describes Entities that can have Actions queued for
// them, and that can get ActionRelated information about those actions.
// TODO(jcw4) consider implementing separate Actor classes for this
// interface; for example UnitActor that implements this interface, and
// takes a Unit and performs all these actions.
type ActionReceiver interface {
	Entity

	// AddAction queues an action with the given name and payload for this
	// ActionReceiver.
	AddAction(name string, payload map[string]interface{}) (*Action, error)

	// CancelAction removes a pending Action from the queue for this
	// ActionReceiver and marks it as cancelled.
	CancelAction(action *Action) (*ActionResult, error)

	// WatchActions returns a StringsWatcher that will notify on changes
	// to the queued actions for this ActionReceiver.
	WatchActions() StringsWatcher

	// Actions returns the list of Actions queued for this ActionReceiver.
	Actions() ([]*Action, error)

	// ActionResults returns the list of completed ActionResults that were
	// queued on this ActionReceiver.
	ActionResults() ([]*ActionResult, error)

	// Name returns the name that will be used to filter actions
	// that are queued for this ActionReceiver.
	Name() string
}

var (
	_ ActionReceiver = (*Unit)(nil)
	// TODO(jcw4) - use when Actions can be queued for Services.
	//_ ActionReceiver = (*Service)(nil)
)

const actionMarker string = "_a_"

type actionNotificationDoc struct {
	// DocId is the composite _id that can be matched by an
	// idPrefixWatcher that is configured to watch for the
	// ActionReceiver Name() which makes up the first part of this
	// composite _id.
	DocId string `bson:"_id"`

	// EnvUUID is the environment identifier.
	EnvUUID string `bson:"env-uuid"`

	// Receiver is the Name of the Unit or any other ActionReceiver for
	// which this notification is queued.
	Receiver string `bson:"receiver"`

	// ActionID is the unique identifier for the Action this notification
	// represents.
	ActionID string `bson:"uuid"`
}

type actionDoc struct {
	// DocId is the key for this document. The structure of the key is
	// a composite of ActionReceiver.ActionKey() and a UUID
	// to facilitate indexing and prefix filtering.
	DocId string `bson:"_id"`

	// EnvUUID is the environment identifier.
	EnvUUID string `bson:"env-uuid"`

	// Receiver is the Name of the Unit or any other ActionReceiver for
	// which this Action is queued.
	Receiver string `bson:"receiver"`

	// UUID is the unique identifier for this instance of this Action,
	// and is encoded in the DocId too.
	UUID string `bson:"uuid"`

	// Name identifies the action that should be run; it should
	// match an action defined by the unit's charm.
	Name string `bson:"name"`

	// Parameters holds the action's parameters, if any; it should validate
	// against the schema defined by the named action in the unit's charm.
	Parameters map[string]interface{} `bson:"parameters"`

	// Enqueued is the time the action was added.
	Enqueued time.Time `bson:"enqueued"`
}

// Action represents an instruction to do some "action" and is expected
// to match an action definition in a charm.
type Action struct {
	st  *State
	doc actionDoc
}

// Id returns the local name of the Action.
func (a *Action) Id() string {
	return a.st.localID(a.doc.DocId)
}

// NotificationId returns the Id used in the notification watcher.
func (a *Action) NotificationId() string {
	return ensureActionMarker(a.Receiver()) + a.Id()
}

// Receiver returns the Name of the ActionReceiver for which this action
// is enqueued.  Usually this is a Unit Name().
func (a *Action) Receiver() string {
	return a.doc.Receiver
}

// UUID returns the unique id of this Action.
func (a *Action) UUID() string {
	return a.doc.UUID
}

// Name returns the name of the action, as defined in the charm.
func (a *Action) Name() string {
	return a.doc.Name
}

// Parameters will contain a structure representing arguments or parameters to
// an action, and is expected to be validated by the Unit using the Charm
// definition of the Action.
func (a *Action) Parameters() map[string]interface{} {
	return a.doc.Parameters
}

// Enqueued returns the time the action was added to state as a pending
// Action.
func (a *Action) Enqueued() time.Time {
	return a.doc.Enqueued
}

// ValidateTag should be called before calls to Tag() or ActionTag(). It verifies
// that the Action can produce a valid Tag.
func (a *Action) ValidateTag() bool {
	_, ok := names.ParseActionTagFromParts(a.Receiver(), a.UUID())
	return ok
}

// Tag implements the Entity interface and returns a names.Tag that
// is a names.ActionTag.
func (a *Action) Tag() names.Tag {
	return a.ActionTag()
}

// ActionTag returns an ActionTag constructed from this action's
// Prefix and Sequence.
func (a *Action) ActionTag() names.ActionTag {
	return names.JoinActionTag(a.Receiver(), a.UUID())
}

// ActionResults is a data transfer object that holds the key Action
// output and results information.
type ActionResults struct {
	Status  ActionStatus           `json:"status"`
	Results map[string]interface{} `json:"results"`
	Message string                 `json:"message"`
}

// Finish removes action from the pending queue and creates an
// ActionResult to capture the output and end state of the action.
func (a *Action) Finish(results ActionResults) (*ActionResult, error) {
	return a.removeAndLog(results.Status, results.Results, results.Message)
}

// removeAndLog takes the action off of the pending queue, and creates
// an actionresult to capture the outcome of the action.
func (a *Action) removeAndLog(finalStatus ActionStatus, results map[string]interface{}, message string) (*ActionResult, error) {
	doc := newActionResultDoc(a, finalStatus, results, message)
	err := a.st.runTransaction([]txn.Op{
		addActionResultOp(a.st, &doc), {
			C:      actionsC,
			Id:     a.doc.DocId,
			Remove: true,
		}, {
			C:      actionNotificationsC,
			Id:     a.st.docID(ensureActionMarker(a.Receiver()) + a.UUID()),
			Remove: true,
		},
	})
	if err != nil {
		return nil, err
	}
	return a.st.ActionResultByTag(a.ActionTag())
}

// newAction builds an Action for the given State and actionDoc.
func newAction(st *State, adoc actionDoc) *Action {
	return &Action{
		st:  st,
		doc: adoc,
	}
}

// newActionDoc builds the actionDoc with the given name and parameters.
func newActionDoc(st *State, ar ActionReceiver, actionName string, parameters map[string]interface{}) (actionDoc, actionNotificationDoc, error) {
	prefix := ensureActionMarker(ar.Name())
	actionId, err := utils.NewUUID()
	if err != nil {
		return actionDoc{}, actionNotificationDoc{}, err
	}
	envuuid := st.EnvironUUID()

	return actionDoc{
			DocId:      st.docID(actionId.String()),
			EnvUUID:    envuuid,
			Receiver:   ar.Name(),
			UUID:       actionId.String(),
			Name:       actionName,
			Parameters: parameters,
			Enqueued:   nowToTheSecond(),
		}, actionNotificationDoc{
			DocId:    st.docID(prefix + actionId.String()),
			EnvUUID:  envuuid,
			Receiver: ar.Name(),
			ActionID: actionId.String(),
		}, nil
}

var ensureActionMarker = ensureSuffixFn(actionMarker)

// actionIdFromTag converts an ActionTag to an actionId.
func actionIdFromTag(tag names.ActionTag) string {
	ptag := tag.PrefixTag()
	if ptag == nil {
		return ""
	}
	return tag.Suffix()
}
