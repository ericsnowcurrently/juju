// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"io"
	"os"
	"time"

	gitjujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/service/common"
)

type BaseSuite struct {
	gitjujutesting.IsolationSuite

	DataDir string
	Conf    *Conf
	Confdir *confDir

	FakeInit  *fakeInit
	FakeFiles *fakeFiles
}

func (s *BaseSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)

	s.DataDir = "/var/lib/juju"
	s.Conf = &Conf{Conf: common.Conf{
		Desc: "a service",
		Cmd:  "spam",
	}}

	// Patch a few things.
	s.FakeInit = &fakeInit{}
	s.FakeFiles = &fakeFiles{}
	s.FakeFiles.File = s.FakeFiles

	s.PatchValue(&newFileOps, func() fileOperations {
		return s.FakeFiles
	})

	name := "jujud-machine-0"
	initDir := s.DataDir + "/init"
	s.Confdir = newConfDir(name, initDir, InitSystemUpstart, nil)
}

// TODO(ericsnow) Use the fake in the testing repo as soon as it lands.

type FakeCallArgs map[string]interface{}

type FakeCall struct {
	FuncName string
	Args     FakeCallArgs
}

type fake struct {
	calls []FakeCall

	Errors []error
}

func (f *fake) err() error {
	if len(f.Errors) == 0 {
		return nil
	}
	err := f.Errors[0]
	f.Errors = f.Errors[1:]
	return err
}

func (f *fake) addCall(funcName string, args FakeCallArgs) {
	f.calls = append(f.calls, FakeCall{
		FuncName: funcName,
		Args:     args,
	})
}

func (f *fake) SetErrors(errors ...error) {
	f.Errors = errors
}

func (f *fake) CheckCalls(c *gc.C, expected []FakeCall) {
	c.Check(f.calls, jc.DeepEquals, expected)
}

// TODO(ericsnow) Move fakeFiles to service/testing.

type fakeFileInfo struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

type fakeFile struct {
	Info fakeFileInfo
}

func newFakeFile(name string, size int64) *fakeFile {
	return &fakeFile{fakeFileInfo{
		Name: name,
		Size: size,
		Mode: 0644,
	}}
}

func newFakeDir(name string) *fakeFile {
	return &fakeFile{fakeFileInfo{
		Name:  name,
		Mode:  0755,
		IsDir: true,
	}}
}

func (ff fakeFile) Name() string {
	return ff.Info.Name
}

func (ff fakeFile) Size() int64 {
	return ff.Info.Size
}

func (ff fakeFile) Mode() os.FileMode {
	return ff.Info.Mode
}

func (ff fakeFile) ModTime() time.Time {
	return ff.Info.ModTime
}

func (ff fakeFile) IsDir() bool {
	return ff.Info.IsDir
}

func (ff fakeFile) Sys() interface{} {
	return nil
}

type fakeFiles struct {
	fake

	Exists     bool
	DirEntries []os.FileInfo
	Data       []byte
	File       io.WriteCloser
	NWritten   int
}

func (ff *fakeFiles) exists(name string) (bool, error) {
	ff.addCall("Exists", FakeCallArgs{
		"name": name,
	})
	return ff.Exists, ff.err()
}

func (ff *fakeFiles) mkdirAll(dirname string, mode os.FileMode) error {
	ff.addCall("MkdirAll", FakeCallArgs{
		"dirname": dirname,
		"mode":    mode,
	})
	return ff.err()
}

func (ff *fakeFiles) readDir(dirname string) ([]os.FileInfo, error) {
	ff.addCall("ReadDir", FakeCallArgs{
		"dirname": dirname,
	})
	return ff.DirEntries, ff.err()
}

func (ff *fakeFiles) readFile(filename string) ([]byte, error) {
	ff.addCall("ReadFile", FakeCallArgs{
		"filename": filename,
	})
	return ff.Data, ff.err()
}

func (ff *fakeFiles) createFile(filename string) (io.WriteCloser, error) {
	ff.addCall("CreateFile", FakeCallArgs{
		"filename": filename,
	})
	return ff.File, ff.err()
}

func (ff *fakeFiles) removeAll(name string) error {
	ff.addCall("RemoveAll", FakeCallArgs{
		"name": name,
	})
	return ff.err()
}

func (ff *fakeFiles) chmod(name string, mode os.FileMode) error {
	ff.addCall("Chmod", FakeCallArgs{
		"name": name,
		"mode": mode,
	})
	return ff.err()
}

// Write Implements io.Writer.
func (ff *fakeFiles) Write(data []byte) (int, error) {
	ff.addCall("Write", FakeCallArgs{
		"data": data,
	})
	return ff.NWritten, ff.err()
}

// Write Implements io.Closer.
func (ff *fakeFiles) Close() error {
	ff.addCall("Close", nil)
	return ff.err()
}

// TODO(ericsnow) Move fakeInit to service/testing.

type fakeInit struct {
	fake

	Names   []string
	Enabled bool
	SInfo   *common.ServiceInfo
	SConf   *common.Conf
	Data    []byte
}

func (fi *fakeInit) List(include ...string) ([]string, error) {
	fi.addCall("List", FakeCallArgs{
		"include": include,
	})
	return fi.Names, fi.err()
}

func (fi *fakeInit) Start(name string) error {
	fi.addCall("Start", FakeCallArgs{
		"name": name,
	})
	return fi.err()
}

func (fi *fakeInit) Stop(name string) error {
	fi.addCall("Stop", FakeCallArgs{
		"name": name,
	})
	return fi.err()
}

func (fi *fakeInit) Enable(name, filename string) error {
	fi.addCall("Enable", FakeCallArgs{
		"name":     name,
		"filename": filename,
	})
	return fi.err()
}

func (fi *fakeInit) Disable(name string) error {
	fi.addCall("Disable", FakeCallArgs{
		"name": name,
	})
	return fi.err()
}

func (fi *fakeInit) IsEnabled(name string, filenames ...string) (bool, error) {
	fi.addCall("IsEnabled", FakeCallArgs{
		"name":      name,
		"filenames": filenames,
	})
	return fi.Enabled, fi.err()
}

func (fi *fakeInit) Info(name string) (*common.ServiceInfo, error) {
	fi.addCall("Info", FakeCallArgs{
		"name": name,
	})
	return fi.SInfo, fi.err()
}

func (fi *fakeInit) Conf(name string) (*common.Conf, error) {
	fi.addCall("Conf", FakeCallArgs{
		"name": name,
	})
	return fi.SConf, fi.err()
}

func (fi *fakeInit) Serialize(conf *common.Conf) ([]byte, error) {
	fi.addCall("Serialize", FakeCallArgs{
		"conf": conf,
	})
	return fi.Data, fi.err()
}

func (fi *fakeInit) Deserialize(data []byte) (*common.Conf, error) {
	fi.addCall("Deserialize", FakeCallArgs{
		"data": data,
	})
	return fi.SConf, fi.err()
}
