// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/backups/config"
	ctesting "github.com/juju/juju/state/backups/config/testing"
	jujutesting "github.com/juju/juju/testing"
	"github.com/juju/juju/utils"
)

type BackupsSuite struct {
	jujutesting.BaseSuite

	WS      utils.Workspace
	RootDir string
	Paths   config.Paths
	DBInfo  config.DBInfo
	Config  config.BackupsConfig

	testFiles  []string
	ranCommand bool
}

func (s *BackupSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)

	s.WS = jujutesting.NewTestWorkspace(c)

	pathsWS, paths := ctesting.NewPaths(c)
	dbInfo := ctesting.NewDBInfo(c)
	s.Config = config.NewBackupsConfig(dbInfo, paths)
	s.Paths = paths
	s.DBInfo = dbInfo
	s.RootDir = pathsWS.RootDir()
}

func (s *createSuite) CopyArchiveFile(c *gc.C, orig io.ReadCloser) string {
	defer orig.Close()

	bkpFile := "juju-backup.tar.gz"
	_, file, _ := s.WS.Create(bkpFile)
	defer file.Close()

	_, err = io.Copy(file, orig)
	c.Assert(err, gc.IsNil)

	return s.WS.Resolve(bkpFile)
}
