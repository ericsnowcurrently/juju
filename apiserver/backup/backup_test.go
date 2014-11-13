// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup_test

import (
	"io"
	"io/ioutil"

	"github.com/juju/errors"
	"github.com/juju/names"
	gc "gopkg.in/check.v1"

	backupAPI "github.com/juju/juju/apiserver/backup"
	"github.com/juju/juju/apiserver/common"
	apiservertesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/backup"
)

type fakeBackups struct {
	meta    *backup.Metadata
	archive io.ReadCloser
	err     error
}

func (i *fakeBackups) Create(backup.Paths, backup.DBInfo, backup.Origin, string) (*backup.Metadata, error) {
	if i.err != nil {
		return nil, errors.Trace(i.err)
	}
	return i.meta, nil
}

func (i *fakeBackups) Get(string) (*backup.Metadata, io.ReadCloser, error) {
	if i.err != nil {
		return nil, nil, errors.Trace(i.err)
	}
	return i.meta, i.archive, nil
}

func (i *fakeBackups) List() ([]backup.Metadata, error) {
	if i.err != nil {
		return nil, errors.Trace(i.err)
	}
	return []backup.Metadata{*i.meta}, nil
}

func (i *fakeBackups) Remove(string) error {
	if i.err != nil {
		return errors.Trace(i.err)
	}
	return nil
}

type backupSuite struct {
	testing.JujuConnSuite
	resources  *common.Resources
	authorizer *apiservertesting.FakeAuthorizer
	api        *backupAPI.API
	meta       *backup.Metadata
}

var _ = gc.Suite(&backupSuite{})

func (s *backupSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	s.resources = common.NewResources()
	s.resources.RegisterNamed("dataDir", common.StringResource("/var/lib/juju"))
	tag := names.NewLocalUserTag("spam")
	s.authorizer = &apiservertesting.FakeAuthorizer{Tag: tag}
	var err error
	s.api, err = backupAPI.NewAPI(s.State, s.resources, s.authorizer)
	c.Assert(err, gc.IsNil)
	s.meta = s.newMeta("")
}

func (s *backupSuite) newMeta(notes string) *backup.Metadata {
	origin := backup.NewOrigin("<env ID>", "<machine ID>", "<hostname>")
	return backup.NewMetadata(origin, notes, nil)
}

func (s *backupSuite) setBackups(c *gc.C, meta *backup.Metadata, err string) *fakeBackups {
	fake := fakeBackups{
		meta: meta,
	}
	if err != "" {
		fake.err = errors.Errorf(err)
	}
	s.PatchValue(backupAPI.NewBackups,
		func(*state.State) (backup.Backups, io.Closer) {
			return &fake, ioutil.NopCloser(nil)
		},
	)
	return &fake
}

func (s *backupSuite) TestRegistered(c *gc.C) {
	_, err := common.Facades.GetType("Backup", 0)
	c.Check(err, gc.IsNil)
}

func (s *backupSuite) TestNewAPIOkay(c *gc.C) {
	_, err := backupAPI.NewAPI(s.State, s.resources, s.authorizer)
	c.Check(err, gc.IsNil)
}

func (s *backupSuite) TestNewAPINotAuthorized(c *gc.C) {
	s.authorizer.Tag = names.NewServiceTag("eggs")
	_, err := backupAPI.NewAPI(s.State, s.resources, s.authorizer)

	c.Check(errors.Cause(err), gc.Equals, common.ErrPerm)
}
