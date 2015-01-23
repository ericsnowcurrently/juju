// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/utils"

	"github.com/juju/juju/service/common"
	"github.com/juju/juju/utils/ssh"
)

type limitedInitSystem interface {
	Serialize(name string, conf *common.Conf) ([]byte, error)
	Commands() *common.InitSystemCommands
}

type RemoteServices struct {
	Username string
	Hostname string
	init     limitedInitSystem
	configs  *serviceConfigs
}

func NewRemoteServices(user, host, dataDir, initSystem string) (*RemoteInitSystem, error) {
	// Get the InitSystem to use.
	newInitSystem, ok := initSystems[initSystem]
	if !ok {
		return nil, errors.NotFoundf("init system %q", initSystem)
	}
	init := newInitSystem()
	if init.Commands() == nil {
		return nil, errors.NotSupportedf("shell commands for init system %q", initSystem)
	}

	// Get the service configs.
	configs := newConfigs(dataDir, initSystem, jujuPrefixes...)
	configs.fops = newCachedFileOperations()

	return &RemoteServices{
		Username: user,
		Hostname: host,
		commands: commands,
		configs:  configs,
	}
}

// TODO(ericsnow) Add support for Windows.
func (rs *RemoteServices) runSSH(command string) (string, error) {
	cmd := ssh.Command(
		rs.Username+"@"+rs.Host,
		[]string{"/bin/bash"},
		nil,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(command)

	// Run the command.
	if err := cmd.Run(); err != nil {
		if stderr.Len() != 0 {
			err = errors.Annotate(err, strings.TrimSpace(stderr.String()))
		} else {
			err = errors.Trace(err)
		}
		return "", err
	}

	// Clean up the output.
	output := strings.TrimSpace(stdout.String())
	return output, nil
}

func (rs *RemoteServices) List() ([]string, error) {
	// TODO(ericsnow) Ditch the try-them-all strategy.

	commands := rs.init.Commands()
	if commands != nil {
		names, err := rs.list(commands)
		return names, errors.Trace(err)
	}

	// Try each of the known init systems.
	// TODO(ericsnow) Is there a better way to do this? Perhaps the
	// caller should responsible for knowing which init system the
	// remote host is using?
	for name, initfunc := range initSystems {
		init := initFunc()
		commands := init.Commands()
		if commands == nil {
			continue
		}

		names, err := rs.list(commands)
		if err != nil {
			continue
		}

		return names, nil
	}

	return nil, errors.Errorf("could not determine init system on %q", ris.Hostname)
}

func (rs *RemoteServices) list(commands common.InitSystemCommands) ([]string, error) {
	command, err := commands.List()
	if err != nil {
		return nil, errors.Trace(err)
	}

	output, err := rs.runSSH(command)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(output) == 0 {
		return nil, errors.NotFoundf("init services on %q", ris.Hostname)
	}

	var names []string
	for _, name := range strings.Split(output, "\n") {
		name = strings.TrimSpace(name)
		if hasPrefix(name, prefixes...) {
			names = append(names, name)
		}
	}
	return names, nil
}

func replayCommands(fops *cachedFileOperations, commands common.InitSystemCommands) ([]string, error) {
    var commands []string
    for _, op := range fops.ops {
        switch op.kind {
        case "exists":
            ...
        case "mkdirs":
            ...
        case "readfile":
            ...
        case "createfile":
            data := fops.files[op.target].Bytes()
            cmd, err := commands.Write(op.target, data)
            if err != nil {
                return nil, errors.Trace(err)
            }
            command = append(commands, cmd)
        case "removeall":
            ...
        case "chmod":
            ...
        default:
            return nil, errors.NotValidf("op kind %q", op.kind)
        }
    }
    return commands, nil
}
