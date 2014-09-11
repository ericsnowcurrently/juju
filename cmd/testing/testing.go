// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/juju/cmd"
	"launchpad.net/gnuflag"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/juju/osenv"
	"github.com/juju/juju/provider/dummy"
	coretesting "github.com/juju/juju/testing"
)

// FlagRunMain is used to indicate that the -run-main flag was used.
var FlagRunMain = flag.Bool("run-main", false, "Run the application's main function for recursive testing")

// BadRun is used to run a command, check the exit code, and return the output.
func BadRun(c *gc.C, exit int, args ...string) string {
	localArgs := append([]string{"-test.run", "TestRunMain", "-run-main", "--"}, args...)
	ps := exec.Command(os.Args[0], localArgs...)
	ps.Env = append(os.Environ(), osenv.JujuHomeEnvKey+"="+osenv.JujuHome())
	output, err := ps.CombinedOutput()
	c.Logf("command output: %q", output)
	if exit != 0 {
		c.Assert(err, gc.ErrorMatches, fmt.Sprintf("exit status %d", exit))
	}
	return string(output)
}

// HelpText returns a command's formatted help text.
func HelpText(command cmd.Command, name string) string {
	buff := &bytes.Buffer{}
	info := command.Info()
	info.Name = name
	f := gnuflag.NewFlagSet(info.Name, gnuflag.ContinueOnError)
	command.SetFlags(f)
	buff.Write(info.Help(f))
	return buff.String()
}

// NullContext returns a no-op command context.
func NullContext(c *gc.C) *cmd.Context {
	ctx, err := cmd.DefaultContext()
	c.Assert(err, gc.IsNil)
	ctx.Stdin = io.LimitReader(nil, 0)
	ctx.Stdout = ioutil.Discard
	ctx.Stderr = ioutil.Discard
	return ctx
}

// RunCommand runs the command and returns channels holding the
// command's operations and errors.
func RunCommand(ctx *cmd.Context, com cmd.Command, args ...string) (opc chan dummy.Operation, errc chan error) {
	if ctx == nil {
		panic("ctx == nil")
	}
	errc = make(chan error, 1)
	opc = make(chan dummy.Operation, 200)
	dummy.Listen(opc)
	go func() {
		// signal that we're done with this ops channel.
		defer dummy.Listen(nil)

		err := coretesting.InitCommand(com, args)
		if err != nil {
			errc <- err
			return
		}

		err = com.Run(ctx)
		errc <- err
	}()
	return
}

// CheckHelp will check the output of run a command with its --help option.
func CheckHelp(c *gc.C, supercmd string, command cmd.Command) {
	info := command.Info()

	_options := ""
	if !command.IsSuperCommand() {
		_options = "[options] "
	}
	_usage := fmt.Sprintf("usage: %s %s %s%s", supercmd, info.Name, _options, info.Args)
	_purpose := fmt.Sprintf("purpose: %s", info.Purpose)

	// Run the command, ensuring it is actually there.
	args := append(strings.Fields(supercmd), info.Name, "--help")
	out := BadRun(c, 0, args...)
	out = out[:len(out)-1] // Strip the trailing \n.

	// Check the usage string.
	parts := strings.SplitN(out, "\n", 2)
	c.Assert(parts, gc.HasLen, 2)
	usage := parts[0]
	out = parts[1]
	c.Check(usage, gc.Equals, _usage)

	// Check the purpose string.
	parts = strings.SplitN(out, "\n\n", 2)
	c.Assert(parts, gc.HasLen, 2)
	purpose := parts[0]
	out = parts[1]
	c.Check(purpose, gc.Equals, _purpose)

	// Check the options.
	parts = strings.SplitN(out, "\n\n", 2)
	c.Assert(parts, gc.HasLen, 2)
	options := strings.Split(parts[0], "\n-")
	out = parts[1]
	c.Assert(options, gc.HasLen, 4)
	c.Check(options[0], gc.Equals, "options:")
	c.Check(strings.Contains(options[1], "environment"), gc.Equals, true)

	// Check the doc.
	doc := out
	c.Check(doc, gc.Equals, info.Doc)
}
