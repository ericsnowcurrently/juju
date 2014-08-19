// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package utils_test

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/testing"
	"github.com/juju/juju/utils"
)

type workspaceSuite struct {
	testing.BaseSuite
	rootDir string
	ws      *utils.WorkspaceTree
}

// Register the suite.
var _ = gc.Suite(&workspaceSuite{})

func (s *workspaceSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)

	rootDir := filepath.Join(c.MkDir(), "spam")
	err := os.Mkdir(rootDir, 0777)
	c.Assert(err, gc.IsNil)
	s.rootDir = rootDir

	ws := utils.NewRootedWorkspace(s.rootDir)
	s.ws = ws
}

func (s *workspaceSuite) checkExists(c *gc.C, path string) {
	_, err := os.Stat(path)
	if err != nil {
		c.Assert(os.IsNotExist(err), gc.Equals, true)
		c.Errorf("%q does not exist", path)
	}
}

func (s *workspaceSuite) checkNotExists(c *gc.C, path string) {
	_, err := os.Stat(path)
	if err != nil {
		c.Assert(os.IsNotExist(err), gc.Equals, true)
		return
	}
	c.Errorf("%q exists", path)
}

func (s *workspaceSuite) TestNewWorkspace(c *gc.C) {
	ws, err := utils.NewWorkspace()
	c.Assert(err, gc.IsNil)

	prefix := filepath.Join(os.TempDir(), "juju-workspace-")
	if c.Check(strings.HasPrefix(ws.RootDir(), prefix), gc.Equals, true) {
		defer os.RemoveAll(ws.RootDir())
	}
	s.checkExists(c, ws.RootDir())
}

func (s *workspaceSuite) TestNewRootedWorkspace(c *gc.C) {
	rootDir := filepath.Join(c.MkDir(), "foo")
	ws := utils.NewRootedWorkspace(rootDir)

	c.Check(ws.RootDir(), gc.Equals, rootDir)
	s.checkNotExists(c, ws.RootDir())
}

func (s *workspaceSuite) TestResolve(c *gc.C) {
	resolved := s.ws.Resolve("eggs")

	c.Check(resolved, gc.Equals, filepath.Join(s.rootDir, "eggs"))
	s.checkNotExists(c, resolved)
}

func (s *workspaceSuite) TestSubOkay(c *gc.C) {
	sub, err := s.ws.Sub("eggs")
	c.Assert(err, gc.IsNil)

	c.Check(sub.RootDir(), gc.Equals, filepath.Join(s.rootDir, "eggs"))
	s.checkExists(c, sub.RootDir())
}

func (s *workspaceSuite) TestSubResolve(c *gc.C) {
	sub, err := s.ws.Sub("eggs")
	c.Assert(err, gc.IsNil)
	resolved := sub.Resolve("ham")

	c.Check(resolved, gc.Equals, filepath.Join(s.rootDir, "eggs", "ham"))
}

func (s *workspaceSuite) TestSubDeep(c *gc.C) {
	sub, err := s.ws.Sub(filepath.Join("eggs", "ham"))
	c.Assert(err, gc.IsNil)

	c.Check(sub.RootDir(), gc.Equals, filepath.Join(s.rootDir, "eggs", "ham"))
	s.checkExists(c, sub.RootDir())
}

func (s *workspaceSuite) TestSubCached(c *gc.C) {
	sub1, err := s.ws.Sub("eggs")
	c.Assert(err, gc.IsNil)
	sub2, err := s.ws.Sub("eggs")
	c.Assert(err, gc.IsNil)

	c.Check(sub1, gc.Equals, sub2)
}

func (s *workspaceSuite) TestSubNested(c *gc.C) {
	sub1, err := s.ws.Sub(filepath.Join("eggs", "ham"))
	c.Assert(err, gc.IsNil)
	sub2, err := s.ws.Sub("eggs")
	c.Assert(err, gc.IsNil)
	sub3, err := sub2.Sub("ham")
	c.Assert(err, gc.IsNil)

	c.Check(sub3.RootDir(), gc.Equals, filepath.Join(s.rootDir, "eggs", "ham"))
	c.Check(sub3, gc.Equals, sub1)
}

func (s *workspaceSuite) TestSubTemp(c *gc.C) {
	sub, err := s.ws.Sub("")
	c.Assert(err, gc.IsNil)

	prefix := filepath.Join(s.rootDir, string(filepath.Separator))
	c.Check(strings.HasPrefix(sub.RootDir(), prefix), gc.Equals, true)
	s.checkExists(c, sub.RootDir())
}

func (s *workspaceSuite) TestMkDirsOkay(c *gc.C) {
	path, err := s.ws.MkDirs(filepath.Join("spam", "eggs"), 0777)
	c.Assert(err, gc.IsNil)

	c.Check(path, gc.Equals, filepath.Join("spam", "eggs"))
	s.checkExists(c, filepath.Join(s.rootDir, "spam", "eggs"))
}

func (s *workspaceSuite) TestMkDirsTemp(c *gc.C) {
	path, err := s.ws.MkDirs("", 0777)
	c.Assert(err, gc.IsNil)

	c.Check(path, gc.Not(jc.Contains), string(filepath.Separator))
	s.checkExists(c, filepath.Join(s.rootDir, path))
}

func (s *workspaceSuite) TestTouchNotExists(c *gc.C) {
	s.checkNotExists(c, s.ws.Resolve("eggs"))
	path, err := s.ws.Touch("eggs")
	c.Assert(err, gc.IsNil)

	c.Check(path, gc.Equals, "eggs")
	s.checkExists(c, s.ws.Resolve("eggs"))
}

func (s *workspaceSuite) TestTouchExists(c *gc.C) {
	_, err := s.ws.Touch("eggs")
	c.Assert(err, gc.IsNil)
	_, err = s.ws.Touch("eggs")
	c.Assert(err, gc.IsNil)

	s.checkExists(c, s.ws.Resolve("eggs"))
}

func (s *workspaceSuite) TestTouchTemp(c *gc.C) {
	path, err := s.ws.Touch("")
	c.Assert(err, gc.IsNil)

	c.Check(path, gc.Not(gc.Equals), "")
	s.checkExists(c, s.ws.Resolve(path))
}

func (s *workspaceSuite) TestCreateOkay(c *gc.C) {
	s.checkNotExists(c, s.ws.Resolve("eggs"))
	path, file, err := s.ws.Create("eggs")
	c.Assert(err, gc.IsNil)
	defer file.Close()

	c.Check(path, gc.Equals, "eggs")
	c.Check(file.Name(), gc.Equals, s.ws.Resolve("eggs"))
	s.checkExists(c, s.ws.Resolve("eggs"))
}

func (s *workspaceSuite) TestCreateExists(c *gc.C) {
	_, err := s.ws.Touch("eggs")
	c.Assert(err, gc.IsNil)
	_, file, err := s.ws.Create("eggs")
	defer file.Close()

	c.Check(errors.Cause(err), jc.Satisfies, os.IsExist)
}

func (s *workspaceSuite) TestCreateTemp(c *gc.C) {
	path, file, err := s.ws.Create("")
	c.Assert(err, gc.IsNil)
	defer file.Close()

	c.Check(path, gc.Not(gc.Equals), "")
	s.checkExists(c, s.ws.Resolve(path))
}

func (s *workspaceSuite) TestOpenOkay(c *gc.C) {
	_, err := s.ws.Touch("eggs")
	c.Assert(err, gc.IsNil)
	file, err := s.ws.Open("eggs")
	defer file.Close()

	c.Check(file.Name(), gc.Equals, s.ws.Resolve("eggs"))
}

func (s *workspaceSuite) TestOpenMissing(c *gc.C) {
	_, err := s.ws.Open("eggs")

	c.Check(err, gc.NotNil)
}

func (s *workspaceSuite) TestCleanUp(c *gc.C) {
	err := s.ws.CleanUp()
	c.Assert(err, gc.IsNil)

	s.checkNotExists(c, s.ws.RootDir())
}
