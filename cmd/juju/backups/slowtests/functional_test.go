// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// TODO(ericsnow) Use build tags:
//  http://golang.org/pkg/go/build/#hdr-Build_Constraints

package slowtests_test

import (
	"os/user"
	"path/filepath"

	"github.com/juju/cmd"
	"github.com/juju/loggo"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/tar"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/params"
	apibackups "github.com/juju/juju/cmd/juju/backups"
	"github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state/backups"
	backupstesting "github.com/juju/juju/state/backups/testing"
	coretesting "github.com/juju/juju/testing"
)

type functionalSuite struct {
	testing.JujuConnSuite

	command *apibackups.Command
}

var _ = gc.Suite(&functionalSuite{})

func (s *functionalSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)

	s.command = apibackups.NewCommand().(*apibackups.Command)
}

func (s *functionalSuite) prepFiles(c *gc.C) {
	paths := &backups.Paths{
		DataDir:        s.DataDir(),
		InitDir:        "",
		LoggingConfDir: "",
		LogsDir:        s.LogDir,
		SSHDir:         "",
	}
	u, err := user.Current()
	c.Assert(err, gc.IsNil)
	username := u.Username
	buf, err := backupstesting.NewFilesBundle(paths, username)
	c.Assert(err, gc.IsNil)
	err = tar.UntarFiles(buf, string(filepath.Separator))
	c.Assert(err, gc.IsNil)
}

func (s *functionalSuite) run(c *gc.C, subcmd string, opts ...string) (string, []loggo.TestLogValues, error) {
	logger := "cmd.juju.backups"
	var testWriter loggo.TestWriter
	loggo.RegisterWriter(logger, &testWriter, loggo.ERROR)
	defer loggo.RemoveWriter(logger)

	opts = append([]string{subcmd}, opts...)
	ctx, err := coretesting.RunCommand(c, s.command, opts...)
	c.Check(coretesting.Stderr(ctx), gc.Equals, "")
	return coretesting.Stdout(ctx), testWriter.Log(), err
}

func (s *functionalSuite) runSuccess(c *gc.C, subcmd string, opts ...string) (string, []loggo.TestLogValues) {
	stdout, log, err := s.run(c, subcmd, opts...)
	c.Assert(err, gc.IsNil)
	return stdout, log
}

func (s *functionalSuite) runFailure(c *gc.C, subcmd string, opts ...string) []loggo.TestLogValues {
	stdout, log, err := s.run(c, subcmd, opts...)
	c.Assert(err, gc.Equals, cmd.ErrSilent)
	c.Check(stdout, gc.Equals, "")
	return log
}

func (s *functionalSuite) checkLog(c *gc.C, log []loggo.TestLogValues, expected ...string) {
	c.Check(log, jc.LogMatches, expected)
}

func (s *functionalSuite) parseCreate(c *gc.C, out string) (string, *params.BackupsMetadataResult) {
	return "", nil
}

func (s *functionalSuite) parseInfo(c *gc.C, out string) params.BackupsMetadataResult {
	return params.BackupsMetadataResult{}
}

func (s *functionalSuite) parseList(c *gc.C, out string) []params.BackupsMetadataResult {
	return nil
}

func (s *functionalSuite) checkMeta(c *gc.C, meta *params.BackupsMetadataResult, partial *params.BackupsMetadataResult) {
}

func (s *functionalSuite) checkFile(c *gc.C, filename string, meta *params.BackupsMetadataResult) {
}

func (s *functionalSuite) TestInitial(c *gc.C) {
	// Verify we start out empty.
	stdout, log := s.runSuccess(c, "list")
	c.Check(stdout, gc.Equals, "")
	s.checkLog(c, log, "(no backups found)")
}

func (s *functionalSuite) TestInfoEmpty(c *gc.C) {
	log := s.runFailure(c, "info", "<some ID>")
	s.checkLog(c, log, `Backup "<some ID>" not found`)
}

func (s *functionalSuite) TestDownloadEmpty(c *gc.C) {
	log := s.runFailure(c, "download", "<some ID>")
	s.checkLog(c, log, `Backup "<some ID>" not found`)
}

func (s *functionalSuite) TestRemoveEmpty(c *gc.C) {
	log := s.runFailure(c, "remove", "<some ID>")
	s.checkLog(c, log, `Backup "<some ID>" not found`)
}

func (s *functionalSuite) TestCreate(c *gc.C) {
	stdout, log := s.runSuccess(c, "create")
	c.Check(log, jc.DeepEquals, nil)
	_, partial := s.parseCreate(c, stdout)
	s.checkMeta(c, nil, partial)
}

func (s *functionalSuite) TestCreateRemoveRoundTrip(c *gc.C) {
	stdout, log := s.runSuccess(c, "create")
	c.Check(log, jc.DeepEquals, nil)
	id, partial := s.parseCreate(c, stdout)
	s.checkMeta(c, nil, partial)

	// Verify it was created properly.

	stdout, log = s.runSuccess(c, "info", id)
	c.Check(log, jc.DeepEquals, nil)
	meta := s.parseInfo(c, stdout)
	s.checkMeta(c, &meta, partial)

	stdout, log = s.runSuccess(c, "list")
	c.Check(log, jc.DeepEquals, nil)
	metaList := s.parseList(c, stdout)
	c.Assert(metaList, gc.HasLen, 1)
	c.Check(metaList[0], gc.DeepEquals, meta)

	downloaded := filepath.Join(c.MkDir(), "juju-backup.tgz")
	s.runSuccess(c, "download", "--filename", downloaded, id)
	s.checkFile(c, downloaded, &meta)

	// Remove and verify.

	s.runSuccess(c, "remove", id)

	stdout, log = s.runSuccess(c, "list")
	c.Check(log, jc.DeepEquals, nil)
	c.Check(stdout, gc.Equals, "(no backups found)\n")

	log = s.runFailure(c, "info", id)
	s.checkLog(c, log, `Backup "<some ID>" not found`)

	log = s.runFailure(c, "download", id)
	s.checkLog(c, log, `Backup "<some ID>" not found`)

	log = s.runFailure(c, "remove", id)
	s.checkLog(c, log, `Backup "<some ID>" not found`)
}

// TODO(ericsnow,perrito666) Add a TestCreateRestoreRoundTrip method.
