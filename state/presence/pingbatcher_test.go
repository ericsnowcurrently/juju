// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package presence_test

import (
	"fmt"
	"time"

	// 	"github.com/juju/errors"
	gitjujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/names.v2"
	// 	worker "gopkg.in/juju/worker.v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	// 	"gopkg.in/tomb.v1"

	"github.com/juju/juju/state/presence"
	"github.com/juju/juju/testing"
)

type PingBatcherSuite struct {
	gitjujutesting.MgoSuite
	testing.BaseSuite
	presence *mgo.Collection
	pings    *mgo.Collection
	modelTag names.ModelTag
}

var _ = gc.Suite(&PingBatcherSuite{})

func (s *PingBatcherSuite) SetUpSuite(c *gc.C) {
	s.BaseSuite.SetUpSuite(c)
	s.MgoSuite.SetUpSuite(c)
	uuid, err := utils.NewUUID()
	c.Assert(err, jc.ErrorIsNil)
	s.modelTag = names.NewModelTag(uuid.String())
}

func (s *PingBatcherSuite) TearDownSuite(c *gc.C) {
	s.MgoSuite.TearDownSuite(c)
	s.BaseSuite.TearDownSuite(c)
}

func (s *PingBatcherSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
	s.MgoSuite.SetUpTest(c)

	db := s.MgoSuite.Session.DB("presence")
	s.presence = db.C("presence")
	s.pings = db.C("presence.pings")

	presence.FakeTimeSlot(0)
}

func (s *PingBatcherSuite) TearDownTest(c *gc.C) {
	s.MgoSuite.TearDownTest(c)
	s.BaseSuite.TearDownTest(c)

	presence.RealTimeSlot()
	presence.RealPeriod()
}

func (s *PingBatcherSuite) TestRecordsPings(c *gc.C) {
	pb := presence.NewPingBatcher(s.pings, time.Second)
	pb.Start()
	defer assertStopped(c, pb)

	// UnixNano time rounded to 30s interval
	slot := int64(1497960150)
	pb.Ping("test-uuid", slot, "0", 8)
	pb.Ping("test-uuid", slot, "0", 16)
	pb.Ping("test-uuid", slot, "1", 128)
	pb.Ping("test-uuid", slot, "1", 1)
	c.Assert(pb.ForceFlush(), jc.ErrorIsNil)
	docId := "test-uuid:1497960150"
	var res bson.M
	c.Assert(s.pings.FindId(docId).One(&res), jc.ErrorIsNil)
	c.Check(res["slot"], gc.Equals, slot)
	c.Check(res["alive"], jc.DeepEquals, bson.M{
		"0": int64(24),
		"1": int64(129),
	})
}

func (s *PingBatcherSuite) TestMultipleUUIDs(c *gc.C) {
	pb := presence.NewPingBatcher(s.pings, time.Second)
	pb.Start()
	defer assertStopped(c, pb)

	// UnixNano time rounded to 30s interval
	slot := int64(1497960150)
	uuid1 := "test-uuid1"
	uuid2 := "test-uuid2"
	pb.Ping(uuid1, slot, "0", 8)
	pb.Ping(uuid2, slot, "0", 8)
	pb.Ping(uuid2, slot, "0", 4)
	c.Assert(pb.ForceFlush(), jc.ErrorIsNil)
	docId1 := fmt.Sprintf("%s:%d", uuid1, slot)
	var res bson.M
	c.Assert(s.pings.FindId(docId1).One(&res), jc.ErrorIsNil)
	c.Check(res["slot"], gc.Equals, slot)
	c.Check(res["alive"], jc.DeepEquals, bson.M{
		"0": int64(8),
	})
	docId2 := fmt.Sprintf("%s:%d", uuid2, slot)
	c.Assert(s.pings.FindId(docId2).One(&res), jc.ErrorIsNil)
	c.Check(res["slot"], gc.Equals, slot)
	c.Check(res["alive"], jc.DeepEquals, bson.M{
		"0": int64(12),
	})
}

func (s *PingBatcherSuite) TestMultipleFlushes(c *gc.C) {
	pb := presence.NewPingBatcher(s.pings, time.Second)
	pb.Start()
	defer assertStopped(c, pb)

	slot := int64(1497960150)
	uuid1 := "test-uuid1"
	pb.Ping(uuid1, slot, "0", 8)
	c.Assert(pb.ForceFlush(), jc.ErrorIsNil)

	docId1 := fmt.Sprintf("%s:%d", uuid1, slot)
	var res bson.M
	c.Assert(s.pings.FindId(docId1).One(&res), jc.ErrorIsNil)
	c.Check(res, gc.DeepEquals, bson.M{
		"_id":  docId1,
		"slot": slot,
		"alive": bson.M{
			"0": int64(8),
		},
	})

	pb.Ping(uuid1, slot, "0", 1024)
	c.Assert(pb.ForceFlush(), jc.ErrorIsNil)
	c.Assert(s.pings.FindId(docId1).One(&res), jc.ErrorIsNil)
	c.Check(res, gc.DeepEquals, bson.M{
		"_id":  docId1,
		"slot": slot,
		"alive": bson.M{
			"0": int64(1032),
		},
	})
}

func (s *PingBatcherSuite) TestMultipleSlots(c *gc.C) {
	pb := presence.NewPingBatcher(s.pings, time.Second)
	pb.Start()
	defer assertStopped(c, pb)

	slot1 := int64(1497960150)
	slot2 := int64(1497960180)
	uuid1 := "test-uuid1"
	pb.Ping(uuid1, slot1, "0", 8)
	pb.Ping(uuid1, slot1, "0", 32)
	pb.Ping(uuid1, slot2, "1", 16)
	pb.Ping(uuid1, slot2, "0", 8)
	c.Assert(pb.ForceFlush(), jc.ErrorIsNil)

	docId1 := fmt.Sprintf("%s:%d", uuid1, slot1)
	var res bson.M
	c.Assert(s.pings.FindId(docId1).One(&res), jc.ErrorIsNil)
	c.Check(res, gc.DeepEquals, bson.M{
		"_id":  docId1,
		"slot": slot1,
		"alive": bson.M{
			"0": int64(40),
		},
	})

	docId2 := fmt.Sprintf("%s:%d", uuid1, slot2)
	c.Assert(s.pings.FindId(docId2).One(&res), jc.ErrorIsNil)
	c.Check(res["slot"], gc.Equals, slot2)
	c.Check(res, gc.DeepEquals, bson.M{
		"_id":  docId2,
		"slot": slot2,
		"alive": bson.M{
			"0": int64(8),
			"1": int64(16),
		},
	})
}

func (s *PingBatcherSuite) TestDocBatchSize(c *gc.C) {
	// We don't want to hit an internal flush
	pb := presence.NewPingBatcher(s.pings, time.Hour)
	pb.Start()
	defer assertStopped(c, pb)

	slotBase := int64(1497960150)
	fieldKey := "0"
	fieldBit := uint64(64)
	// 100 slots * 100 models should be 10,000 docs that we are inserting.
	// mgo.Bulk fails if you try to do more than 1000 requests at once, so this would trigger it if we didn't batch properly.
	for modelCounter := 0; modelCounter  < 100; modelCounter ++ {
		for slotOffset := 0; slotOffset < 100; slotOffset++ {
			slot := slotBase + int64(slotOffset * 30)
			uuid := fmt.Sprintf("uuid-%d", modelCounter)
			c.Assert(pb.Ping(uuid, slot, fieldKey, fieldBit), jc.ErrorIsNil)
		}
	}
	c.Assert(pb.ForceFlush(), jc.ErrorIsNil)
	count, err := s.pings.Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Check(count, gc.Equals, 100*100)
}
