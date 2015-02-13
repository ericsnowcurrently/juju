// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package paths_test

import (
	"fmt"
	"os"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/juju/paths"
	"github.com/juju/juju/testing"
)

var _ = gc.Suite(&mongoSuite{})

type mongoSuite struct {
	testing.BaseSuite
}

func (s *mongoSuite) TestFindMongoRestorePathDefaultExists(c *gc.C) {
	calledWithPaths := []string{}
	osStat := func(aPath string) (os.FileInfo, error) {
		calledWithPaths = append(calledWithPaths, aPath)
		return nil, nil
	}
	s.PatchValue(paths.OsStat, osStat)

	jujuRestore := paths.NewMongo().RestorePath()
	mongoPath, err := paths.Find(jujuRestore)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(mongoPath, gc.Equals, "/usr/lib/juju/bin/mongorestore")
	c.Assert(calledWithPaths, gc.DeepEquals, []string{"/usr/lib/juju/bin/mongorestore"})
}

func (s *mongoSuite) TestFindMongoRestorePathDefaultNotExists(c *gc.C) {
	calledWithPaths := []string{}
	osStat := func(aPath string) (os.FileInfo, error) {
		calledWithPaths = append(calledWithPaths, aPath)
		return nil, fmt.Errorf("sorry no mongo")
	}
	s.PatchValue(paths.OsStat, osStat)

	calledWithLookup := []string{}
	execLookPath := func(aLookup string) (string, error) {
		calledWithLookup = append(calledWithLookup, aLookup)
		return "/a/fake/mongo/path", nil
	}
	s.PatchValue(paths.ExecLookPath, execLookPath)

	jujuRestore := paths.NewMongo().RestorePath()
	mongoPath, err := paths.Find(jujuRestore)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(mongoPath, gc.Equals, "/a/fake/mongo/path")
	c.Assert(calledWithPaths, gc.DeepEquals, []string{"/usr/lib/juju/bin/mongorestore"})
	c.Assert(calledWithLookup, gc.DeepEquals, []string{"mongorestore"})
}
