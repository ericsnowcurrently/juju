// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import (
	"time"

	"github.com/juju/errors"
	"github.com/juju/names"
	gitjujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/mongo"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/backups"
	backupstesting "github.com/juju/juju/state/backups/testing"
	statetesting "github.com/juju/juju/state/testing"
	"github.com/juju/juju/testing"
)

type storageSuite struct {
	gitjujutesting.MgoSuite
	backupstesting.BaseSuite
	st *state.State
}

var _ = gc.Suite(&storageSuite{})

func (s *storageSuite) SetUpSuite(c *gc.C) {
	s.BaseSuite.SetUpSuite(c)
	s.MgoSuite.SetUpSuite(c)
}

func (s *storageSuite) TearDownSuite(c *gc.C) {
	s.MgoSuite.TearDownSuite(c)
	s.BaseSuite.TearDownSuite(c)
}

func (s *storageSuite) SetUpTest(c *gc.C) {
	s.MgoSuite.SetUpTest(c)
	s.BaseSuite.SetUpTest(c)

	owner := names.NewLocalUserTag("test-admin")
	mgoInfo := &mongo.MongoInfo{
		Info: mongo.Info{
			Addrs:  []string{gitjujutesting.MgoServer.Addr()},
			CACert: testing.CACert,
		},
	}
	cfg := testing.EnvironConfig(c)
	dialOpts := mongo.DialOpts{
		Timeout: testing.LongWait,
	}
	policy := statetesting.MockPolicy{}
	st, err := state.Initialize(owner, mgoInfo, cfg, dialOpts, &policy)
	c.Assert(err, gc.IsNil)
	s.st = st
}

func (s *storageSuite) TearDownTest(c *gc.C) {
	if s.st != nil {
		s.st.Close()
	}
	s.BaseSuite.TearDownTest(c)
	s.MgoSuite.TearDownTest(c)
}

func (s *storageSuite) metadata(c *gc.C) *backups.Metadata {
	meta := backups.NewMetadata()
	meta.Origin.Environment = s.st.EnvironUUID()
	meta.Origin.Machine = "0"
	meta.Origin.Hostname = "localhost"
	err := backups.UpdateMetadata(meta, int64(42), "some hash")
	c.Assert(err, gc.IsNil)
	return meta
}

func (s *storageSuite) checkMeta(c *gc.C, meta, expected *backups.Metadata, id string,
) {
	if id != "" {
		c.Check(meta.ID(), gc.Equals, id)
	}
	c.Check(meta.Notes, gc.Equals, expected.Notes)
	c.Check(meta.Started.Unix(), gc.Equals, expected.Started.Unix())
	c.Check(meta.Checksum(), gc.Equals, expected.Checksum())
	c.Check(meta.ChecksumFormat(), gc.Equals, expected.ChecksumFormat())
	c.Check(meta.Size(), gc.Equals, expected.Size())
	c.Check(meta.Origin.Environment, gc.Equals, expected.Origin.Environment)
	c.Check(meta.Origin.Machine, gc.Equals, expected.Origin.Machine)
	c.Check(meta.Origin.Hostname, gc.Equals, expected.Origin.Hostname)
	c.Check(meta.Origin.Version, gc.Equals, expected.Origin.Version)
	if meta.Stored() != nil && expected.Stored != nil {
		c.Check(meta.Stored().Unix(), gc.Equals, expected.Stored().Unix())
	} else {
		c.Check(meta.Stored(), gc.Equals, expected.Stored())
	}
}

func (s *storageSuite) TestNewBackupID(c *gc.C) {
	meta := backups.NewMetadata()
	meta.Origin.Environment = "spam"
	meta.Origin.Machine = "0"
	meta.Origin.Hostname = "localhost"
	meta.Started = time.Date(2014, time.Month(9), 12, 13, 19, 27, 0, time.UTC)
	id := backups.NewBackupID(meta)

	c.Check(id, gc.Equals, "20140912-131927.spam")
}

func (s *storageSuite) TestGetBackupMetadataFound(c *gc.C) {
	original := s.metadata(c)
	id, err := backups.AddBackupMetadata(s.st, original)
	c.Assert(err, gc.IsNil)

	meta, err := backups.GetBackupMetadata(s.st, id)
	c.Assert(err, gc.IsNil)

	s.checkMeta(c, meta, original, id)
}

func (s *storageSuite) TestGetBackupMetadataNotFound(c *gc.C) {
	_, err := backups.GetBackupMetadata(s.st, "spam")

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *storageSuite) TestAddBackupMetadataSuccess(c *gc.C) {
	original := s.metadata(c)
	id, err := backups.AddBackupMetadata(s.st, original)
	c.Assert(err, gc.IsNil)

	meta, err := backups.GetBackupMetadata(s.st, id)
	c.Assert(err, gc.IsNil)

	s.checkMeta(c, meta, original, id)
}

func (s *storageSuite) TestAddBackupMetadataGeneratedID(c *gc.C) {
	original := s.metadata(c)
	original.SetID("spam")
	id, err := backups.AddBackupMetadata(s.st, original)
	c.Assert(err, gc.IsNil)

	c.Check(id, gc.Not(gc.Equals), "spam")
}

func (s *storageSuite) TestAddBackupMetadataEmpty(c *gc.C) {
	original := backups.NewMetadata()
	_, err := backups.AddBackupMetadata(s.st, original)

	c.Check(err, gc.NotNil)
}

func (s *storageSuite) TestAddBackupMetadataAlreadyExists(c *gc.C) {
	original := s.metadata(c)
	id, err := backups.AddBackupMetadata(s.st, original)
	c.Assert(err, gc.IsNil)
	err = backups.AddBackupMetadataID(s.st, original, id)

	c.Check(err, jc.Satisfies, errors.IsAlreadyExists)
}

func (s *storageSuite) TestSetBackupStoredSuccess(c *gc.C) {
	stored := time.Now()
	original := s.metadata(c)
	id, err := backups.AddBackupMetadata(s.st, original)
	c.Assert(err, gc.IsNil)
	meta, err := backups.GetBackupMetadata(s.st, id)
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Stored(), gc.IsNil)

	err = backups.SetBackupStored(s.st, id, stored)
	c.Assert(err, gc.IsNil)

	meta, err = backups.GetBackupMetadata(s.st, id)
	c.Assert(err, gc.IsNil)
	c.Check(meta.Stored().Unix(), gc.Equals, stored.UTC().Unix())
}

func (s *storageSuite) TestSetBackupStoredNotFound(c *gc.C) {
	stored := time.Now()
	err := backups.SetBackupStored(s.st, "spam", stored)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *storageSuite) TestNewMongoConnInfoOkay(c *gc.C) {
	tag, err := names.ParseTag("machine-0")
	c.Assert(err, gc.IsNil)
	mgoInfo := mongo.MongoInfo{
		Info: mongo.Info{
			Addrs: []string{"localhost:8080"},
		},
		Tag:      tag,
		Password: "eggs",
	}

	connInfo := backups.NewMongoConnInfo(&mgoInfo)
	err = connInfo.Validate()
	c.Assert(err, gc.IsNil)

	c.Check(connInfo.Address, gc.Equals, "localhost:8080")
	c.Check(connInfo.Username, gc.Equals, "machine-0")
	c.Check(connInfo.Password, gc.Equals, "eggs")
}

func (s *storageSuite) TestNewMongoConnInfoMissingTag(c *gc.C) {
	mgoInfo := mongo.MongoInfo{
		Info: mongo.Info{
			Addrs: []string{"localhost:8080"},
		},
		Password: "eggs",
	}

	connInfo := backups.NewMongoConnInfo(&mgoInfo)
	err := connInfo.Validate()

	c.Check(err, gc.ErrorMatches, "missing username")
}
