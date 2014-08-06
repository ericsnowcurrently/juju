// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state_test

import (
	"time"

	jc "github.com/juju/testing/checkers"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/txn"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state"
)

type upgradesSuite struct {
	BaseStateSuite
}

var _ = gc.Suite(&upgradesSuite{})

func (s *upgradesSuite) TestLastLoginMigrate(c *gc.C) {
	now := time.Now().UTC().Round(time.Second)
	userId := "foobar"
	oldDoc := bson.M{
		"_id_":           userId,
		"_id":            userId,
		"displayname":    "foo bar",
		"deactivated":    false,
		"passwordhash":   "hash",
		"passwordsalt":   "salt",
		"createdby":      "creator",
		"datecreated":    now,
		"lastconnection": now,
	}

	ops := []txn.Op{
		txn.Op{
			C:      "users",
			Id:     userId,
			Assert: txn.DocMissing,
			Insert: oldDoc,
		},
	}
	err := state.RunTransaction(s.State, ops)
	c.Assert(err, gc.IsNil)

	err = state.MigrateUserLastConnectionToLastLogin(s.State)
	c.Assert(err, gc.IsNil)
	user, err := s.State.User(userId)
	c.Assert(err, gc.IsNil)
	c.Assert(*user.LastLogin(), gc.Equals, now)

	// check to see if _id_ field is removed
	userMap := map[string]interface{}{}
	users, closer := state.GetCollection(s.State, "users")
	defer closer()
	err = users.Find(bson.D{{"_id", userId}}).One(&userMap)
	c.Assert(err, gc.IsNil)
	_, keyExists := userMap["_id_"]
	c.Assert(keyExists, jc.IsFalse)
}
