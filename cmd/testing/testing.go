// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/cmd/cmdtesting"
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
func CheckHelp(c *gc.C, command cmd.Command, supercmd string) {
	info := command.Info()

	// Run the command, ensuring it is actually there.
	args := append(strings.Fields(supercmd), info.Name, "--help")
	out := BadRun(c, 0, args...)
	out = out[:len(out)-1] // Strip the trailing \n.

	cmdtesting.CheckHelpOutput(c, out, command, supercmd)
}
