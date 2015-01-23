// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"fmt"
	"path/filepath"

	"github.com/juju/errors"

	"github.com/juju/juju/service/common"
)

func ShutdownCommands(svc *RemoteServices, filename string) ([]string, error) {

}

func CloudInitAgentCommands(svc *RemoteServices, spec *AgentService) ([]string, error) {
	spec.initSystem = svc.initname
	conf := spec.Conf()

	return CloudInitInstallCommands(spec.Name, svc, conf)
}

func CloudInitInstallCommands(name string, svc *RemoteServices, conf *common.Conf) ([]string, error) {
	commands := svc.init.Commands()

	if err := svc.configs.add(name, conf, svc.init); err != nil {
		return nil, errors.Trace(err)
	}

	// TODO(ericsnow) Is there a better way to write the script out
	// to a remote host in cloudinit?
	cmds, err := replayCommands(svc.config.fops, commands)
	if err != nil {
		return nil, errors.Trace(err)
	}

	enable, err := commands.Enable(name, filename)
	if err != nil {
		return nil, errors.Trace(err)
	}
	cmds = append(cmds, enable)

	start, err := commands.Start(name)
	if err != nil {
		return nil, errors.Trace(err)
	}
	cmds = append(cmds, start)

	return cmds, nil
}
