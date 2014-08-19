// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"os"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/utils"
)

type testWorkspace struct {
	utils.WorkspaceTree
	c *gc.C
}

var _ = utils.Workspace((*testWorkspace)(nil))

func NewTestWorkspace(c *gc.C) *testWorkspace {
	ws := utils.NewRootedWorkspace(c.MkDir())
	return &testWorkspace{*ws, c}
}

func (ws *testWorkspace) Sub(dirName string) (utils.Workspace, error) {
	sub, err := ws.WorkspaceTree.Sub(dirName)
	ws.c.Assert(err, gc.IsNil)
	tree := sub.(*utils.WorkspaceTree)
	return &testWorkspace{*tree, ws.c}, nil
}

func (ws *testWorkspace) MkDirs(path string, mode os.FileMode) (string, error) {
	path, err := ws.WorkspaceTree.MkDirs(path, mode)
	ws.c.Assert(err, gc.IsNil)
	return path, nil
}

func (ws *testWorkspace) Touch(filename string) (string, error) {
	filename, err := ws.WorkspaceTree.Touch(filename)
	ws.c.Assert(err, gc.IsNil)
	return filename, nil
}

func (ws *testWorkspace) Create(filename string) (string, *os.File, error) {
	filename, file, err := ws.WorkspaceTree.Create(filename)
	ws.c.Assert(err, gc.IsNil)
	return filename, file, nil
}

func (ws *testWorkspace) Open(filename string) (*os.File, error) {
	file, err := ws.WorkspaceTree.Open(filename)
	ws.c.Assert(err, gc.IsNil)
	return file, nil
}

func (ws *testWorkspace) CleanUp() error {
	err := ws.WorkspaceTree.CleanUp()
	ws.c.Assert(err, gc.IsNil)
	return nil
}
