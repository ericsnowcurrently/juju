// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd_test

import (
	"fmt"

	"github.com/coreos/go-systemd/dbus"
	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils"
	"github.com/juju/utils/fs"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/service/initsystems"
	"github.com/juju/juju/service/initsystems/systemd"
	coretesting "github.com/juju/juju/testing"
)

const confStr = `
[Unit]
Description=juju agent for %s
After=syslog.target
After=network.target
After=systemd-user-sessions.service

[Service]
Type=forking
ExecStart=jujud %s
RemainAfterExit=yes
Restart=always

[Install]
WantedBy=multi-user.target

`

type stubDbusApi struct {
	*testing.Stub

	units []dbus.UnitStatus
}

func (fda *stubDbusApi) addUnit(name, desc, status string) {
	active := ""
	load := "loaded"
	switch status {
	case initsystems.StatusRunning:
		active = "active"
	case initsystems.StatusStopped:
		active = "inactive"
	case initsystems.StatusEnabled:
		active = "inactive"
	case initsystems.StatusError:
		load = "error"
	}

	unit := dbus.UnitStatus{
		Name:        name,
		Description: desc,
		ActiveState: active,
		LoadState:   load,
	}
	fda.units = append(fda.units, unit)
}

func (fda *stubDbusApi) ListUnits() ([]dbus.UnitStatus, error) {
	fda.Stub.AddCall("ListUnits", nil)

	return fda.units, fda.NextErr()
}

func (fda *stubDbusApi) StartUnit(name string, mode string, ch chan<- string) (int, error) {
	fda.Stub.AddCall("StartUnit", name, mode, ch)

	return 0, fda.NextErr()
}

func (fda *stubDbusApi) StopUnit(name string, mode string, ch chan<- string) (int, error) {
	fda.Stub.AddCall("StopUnit", name, mode, ch)

	return 0, fda.NextErr()
}

func (fda *stubDbusApi) EnableUnitFiles(files []string, runtime bool, force bool) (bool, []dbus.EnableUnitFileChange, error) {
	fda.Stub.AddCall("EnableUnitFiles", files, runtime, force)

	return false, nil, fda.NextErr()
}

func (fda *stubDbusApi) DisableUnitFiles(files []string, runtime bool) ([]dbus.DisableUnitFileChange, error) {
	fda.Stub.AddCall("DisableUnitFiles", []interface{}{
		files,
		runtime,
	})

	return nil, fda.NextErr()
}

func (fda *stubDbusApi) Close() {
	fda.Stub.AddCall("Close", nil)

	fda.Stub.NextErr()
}

type initSystemSuite struct {
	coretesting.BaseSuite

	initDir string
	conf    initsystems.Conf
	confStr string

	ch   chan string
	stub *testing.Stub
	conn *stubDbusApi
	fops *fs.StubOps
	init initsystems.InitSystem
}

var _ = gc.Suite(&initSystemSuite{})

func (s *initSystemSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)

	s.ch = make(chan string, 1)
	s.stub = &testing.Stub{}
	s.conn = &stubDbusApi{Stub: s.stub}
	s.fops = &fs.StubOps{Stub: s.stub}
	s.init = systemd.NewSystemd(s.conn, s.fops, s.ch)
	s.conf = initsystems.Conf{
		Desc: "juju agent for machine-0",
		Cmd:  "jujud machine-0",
	}
	s.confStr = s.newConfStr("jujud-machine-0")

	s.PatchValue(&initsystems.RetryAttempts, utils.AttemptStrategy{})
}

func (s *initSystemSuite) newConfStr(name string) string {
	tag := name[len("jujud-"):]
	return fmt.Sprintf(confStr[1:], tag, tag)
}

func (s *initSystemSuite) addUnit(name, status string) {
	tag := name[len("jujud-"):]
	desc := "juju agent for " + tag
	s.conn.addUnit(name, desc, status)
}

func (s *initSystemSuite) TestInitSystemName(c *gc.C) {
	name := s.init.Name()

	c.Check(name, gc.Equals, "systemd")
}

func (s *initSystemSuite) TestInitSystemList(c *gc.C) {
	s.conn.addUnit("jujud-machine-0", "<>", initsystems.StatusRunning)
	s.conn.addUnit("something-else", "<>", initsystems.StatusError)
	s.conn.addUnit("jujud-unit-wordpress-0", "<>", initsystems.StatusRunning)
	s.conn.addUnit("another", "<>", initsystems.StatusStopped)

	names, err := s.init.List()
	c.Assert(err, jc.ErrorIsNil)

	c.Check(names, jc.SameContents, []string{
		"jujud-machine-0",
		"something-else",
		"jujud-unit-wordpress-0",
		"another",
	})
}

func (s *initSystemSuite) TestInitSystemListLimited(c *gc.C) {
	s.conn.addUnit("jujud-machine-0", "<>", initsystems.StatusRunning)
	s.conn.addUnit("something-else", "<>", initsystems.StatusError)
	s.conn.addUnit("jujud-unit-wordpress-0", "<>", initsystems.StatusRunning)
	s.conn.addUnit("another", "<>", initsystems.StatusStopped)

	names, err := s.init.List("jujud-machine-0")
	c.Assert(err, jc.ErrorIsNil)

	c.Check(names, jc.SameContents, []string{"jujud-machine-0"})
}

func (s *initSystemSuite) TestInitSystemListLimitedEmpty(c *gc.C) {
	names, err := s.init.List("jujud-machine-0")
	c.Assert(err, jc.ErrorIsNil)

	c.Check(names, jc.SameContents, []string{})
}

func (s *initSystemSuite) TestInitSystemStart(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusStopped)
	s.ch <- "done"

	err := s.init.Start(name)
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCallNames(c, "ListUnits", "Close", "StartUnit", "Close")
}

func (s *initSystemSuite) TestInitSystemStartAlreadyRunning(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusRunning)

	err := s.init.Start(name)

	c.Check(err, jc.Satisfies, errors.IsAlreadyExists)
}

func (s *initSystemSuite) TestInitSystemStartNotEnabled(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	err := s.init.Start(name)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *initSystemSuite) TestInitSystemStop(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusRunning)
	s.ch <- "done"

	err := s.init.Stop(name)
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCallNames(c, "ListUnits", "Close", "StopUnit", "Close")
}

func (s *initSystemSuite) TestInitSystemStopNotRunning(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusStopped)

	err := s.init.Stop(name)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *initSystemSuite) TestInitSystemStopNotEnabled(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	err := s.init.Stop(name)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *initSystemSuite) TestInitSystemEnable(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	filename := "/var/lib/juju/init/" + name + "/systemd.conf"
	err := s.init.Enable(name, filename)
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCalls(c, []testing.StubCall{{
		FuncName: "ListUnits",
	}, {
		FuncName: "Close",
	}, {
		FuncName: "EnableUnitFiles",
		Args: []interface{}{
			[]string{filename},
			false,
			true,
		},
	}, {
		FuncName: "Close",
	}})
}

func (s *initSystemSuite) TestInitSystemEnableAlreadyEnabled(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusEnabled)

	filename := "/var/lib/juju/init/" + name + "/systemd.conf"
	err := s.init.Enable(name, filename)

	c.Check(err, jc.Satisfies, errors.IsAlreadyExists)
}

func (s *initSystemSuite) TestInitSystemDisable(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusEnabled)

	err := s.init.Disable(name)
	c.Assert(err, jc.ErrorIsNil)

	filename := "/var/lib/juju/init/" + name + "/systemd.conf"
	s.stub.CheckCalls(c, []testing.StubCall{{
		FuncName: "ListUnits",
	}, {
		FuncName: "Close",
	}, {
		FuncName: "DisableUnitFiles",
		Args: []interface{}{
			[]string{filename},
			false,
		},
	}, {
		FuncName: "Close",
	}})
}

func (s *initSystemSuite) TestInitSystemDisableNotEnabled(c *gc.C) {
	name := "jujud-unit-wordpress-0"

	err := s.init.Disable(name)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *initSystemSuite) TestInitSystemIsEnabledTrue(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusEnabled)

	enabled, err := s.init.IsEnabled(name)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(enabled, jc.IsTrue)

	s.stub.CheckCalls(c, []testing.StubCall{{
		FuncName: "ListUnits",
	}, {
		FuncName: "Close",
	}})
}

func (s *initSystemSuite) TestInitSystemIsEnabledFalse(c *gc.C) {
	name := "jujud-unit-wordpress-0"

	enabled, err := s.init.IsEnabled(name)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(enabled, jc.IsFalse)
}

func (s *initSystemSuite) TestInitSystemInfoRunning(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusRunning)

	info, err := s.init.Info(name)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(info, jc.DeepEquals, &initsystems.ServiceInfo{
		Name:        name,
		Description: "juju agent for unit-wordpress-0",
		Status:      initsystems.StatusRunning,
	})

	s.stub.CheckCalls(c, []testing.StubCall{{
		FuncName: "ListUnits",
	}, {
		FuncName: "Close",
	}})
}

func (s *initSystemSuite) TestInitSystemInfoNotRunning(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusStopped)

	info, err := s.init.Info(name)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(info, jc.DeepEquals, &initsystems.ServiceInfo{
		Name:        name,
		Description: "juju agent for unit-wordpress-0",
		Status:      initsystems.StatusStopped,
	})
}

func (s *initSystemSuite) TestInitSystemInfoNotEnabled(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	_, err := s.init.Info(name)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *initSystemSuite) TestInitSystemConf(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	s.addUnit(name, initsystems.StatusEnabled)

	conf, err := s.init.Conf(name)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(conf, jc.DeepEquals, &initsystems.Conf{
		Desc: `juju agent for unit-wordpress-0`,
		Cmd:  "jujud unit-wordpress-0",
	})

	s.stub.CheckCalls(c, []testing.StubCall{{
		FuncName: "ListUnits",
	}, {
		FuncName: "Close",
	}})
}

func (s *initSystemSuite) TestInitSystemConfNotEnabled(c *gc.C) {
	name := "jujud-unit-wordpress-0"

	_, err := s.init.Conf(name)

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *initSystemSuite) TestInitSystemValidate(c *gc.C) {
	confName, err := s.init.Validate("jujud-machine-0", s.conf)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(confName, gc.Equals, "jujud-machine-0.service")
	s.stub.CheckCalls(c, nil)
}

func (s *initSystemSuite) TestInitSystemValidateFull(c *gc.C) {
	s.conf.Env = map[string]string{
		"x": "y",
	}
	s.conf.Limit = map[string]string{
		"nofile": "10",
	}
	s.conf.Out = "syslog"

	_, err := s.init.Validate("jujud-machine-0", s.conf)
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCalls(c, nil)
}

func (s *initSystemSuite) TestInitSystemValidateInvalid(c *gc.C) {
	s.conf.Cmd = ""

	_, err := s.init.Validate("jujud-machine-0", s.conf)

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *initSystemSuite) TestInitSystemValidateInvalidOut(c *gc.C) {
	s.conf.Out = "/var/log/juju/machine-0.log"

	_, err := s.init.Validate("jujud-machine-0", s.conf)

	expected := errors.NotValidf("Out")
	c.Check(errors.Cause(err), gc.FitsTypeOf, expected)
}

func (s *initSystemSuite) TestInitSystemValidateInvalidLimit(c *gc.C) {
	s.conf.Limit = map[string]string{
		"x": "y",
	}

	_, err := s.init.Validate("jujud-machine-0", s.conf)

	expected := errors.NotValidf("Limit")
	c.Check(errors.Cause(err), gc.FitsTypeOf, expected)
}

func (s *initSystemSuite) TestInitSystemSerialize(c *gc.C) {
	data, err := s.init.Serialize("jujud-machine-0", s.conf)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(string(data), gc.Equals, s.confStr)

	s.stub.CheckCalls(c, nil)
}

func (s *initSystemSuite) TestInitSystemSerializeUnsupported(c *gc.C) {
	tag := "unit-wordpress-0"
	name := "jujud-unit-wordpress-0"
	conf := initsystems.Conf{
		Desc: "juju agent for " + tag,
		Cmd:  "jujud " + tag,
		Out:  "/var/log/juju/" + tag,
	}
	_, err := s.init.Serialize(name, conf)

	expected := errors.NotValidf("Out")
	c.Check(errors.Cause(err), gc.FitsTypeOf, expected)
}

func (s *initSystemSuite) TestInitSystemDeserialize(c *gc.C) {
	name := "jujud-unit-wordpress-0"
	data := s.newConfStr(name)
	conf, err := s.init.Deserialize([]byte(data), name)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(conf, jc.DeepEquals, &initsystems.Conf{
		Desc: "juju agent for unit-wordpress-0",
		Cmd:  "jujud unit-wordpress-0",
	})

	s.stub.CheckCalls(c, nil)
}

func (s *initSystemSuite) TestInitSystemDeserializeUnsupported(c *gc.C) {
	name := "jujud-machine-0"
	data := `
[Unit]
Description=juju agent for machine-0
After=syslog.target
After=network.target
After=systemd-user-sessions.service

[Service]
Type=forking
StandardOutput=/var/log/juju/machine-0.log
ExecStart=jujud machine-0
RemainAfterExit=yes
Restart=always

[Install]
WantedBy=multi-user.target

`[1:]
	_, err := s.init.Deserialize([]byte(data), name)

	expected := errors.NotValidf("Out")
	c.Check(errors.Cause(err), gc.FitsTypeOf, expected)
}
