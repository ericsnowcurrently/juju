// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package agent

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/juju/errors"

	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/tools"
	"github.com/juju/juju/version"
)

type UpgradeContext struct {
	Client version.Number
	Agent  version.Number
	//Available []version.Number
	Tools tools.List
}

type UpgradeClient interface {
	EnvClient
}

func NewUpgradeContext(client UpgradeClient, allTools ...tools.Tools) UpgradeContext {
	ctx := UpgradeContext{
		Client: version.Current.Number,
		Tools:  tools.List(allTools),
	}

	// Pull the agent version, if a client is available.
	agentVersion, err := getEnvVersion(client)
	if err != nil {
		return ctx, errors.Trace(err)
	}
	ctx.Agent = agentVersion

	return ctx, nil
}

func (ctx UpgradeContext) Validate() error {
	// Do not allow an upgrade if the agent is on a greater major version than the CLI.
	if ctx.Agent.Major > ctx.Client.Major {
		return errors.NewNotValid(nil, fmt.Sprintf("cannot upgrade a %s environment with a %s client", ctx.Agent, ctx.Client))
	}

	if len(ctx.Tools) == 0 {
		return errors.NewNotValid(nil, "missing available tools")
	}

	return nil
}

//func (ctx UpgradeContext) LoadTools(target version.Number) version.Number {
//    ListTools(client, major)
//    UploadTools(client, target)
//}

func (ctx UpgradeContext) DecideTarget(target version.Number) version.Number {
	switch {
	case target == version.Zero:
		if ctx.Client == version.Zero {
			logger.Warningf("Client is not set.")
		}
		return ctx.Client
	default:
		return target
	}
}

func (ctx UpgradeContext) IsUpToDate(target version.Number) bool {
	if target == ctx.Agent {
		if ctx.Agent == version.Zero {
			logger.Warningf("Client is not set.")
		}
		return true
	}
	return false
}

func (ctx UpgradeContext) ResolveTarget() (version.Number, error) {
	// No explicitly specified version, so find the version to which we
	// need to upgrade. If the CLI and agent major versions match, we find
	// next available stable release to upgrade to by incrementing the
	// minor version, starting from the current agent version and doing
	// major.minor+1.patch=0. If the CLI has a greater major version,
	// we just use the CLI version as is.
	resolved := ctx.Agent
	if resolved.Major == ctx.Client.Major {
		resolved.Minor += 1
		resolved.Patch = 0
		// Explicitly skip 1.21 and 1.23.
		if resolved.Major == 1 && (resolved.Minor == 21 || resolved.Minor == 23) {
			resolved.Minor += 1
		}
	} else {
		resolved = ctx.Client
	}

	newestNextStable, found := context.tools.NewestCompatible(resolved)
	if found {
		logger.Debugf("found a more recent stable version %s", newestNextStable)
		context.chosen = newestNextStable
	} else {
		newestCurrent, found := context.tools.NewestCompatible(context.agent)
		if found {
			logger.Debugf("found more recent current version %s", newestCurrent)
			context.chosen = newestCurrent
		} else {
			if context.agent.Major != context.client.Major {
				return fmt.Errorf("no compatible tools available")
			} else {
				return fmt.Errorf("no more recent supported versions available")
			}
		}
	}

	return resolved, nil
}

func (ctx UpgradeContext) MatchTarget(target version.Number) (version.Number, tools.List, error) {
	// If not completely specified already, pick a single tools version.
	filtered, err := ctx.Tools.Match(tools.Filter{
		Number: target,
	})
	if err != nil {
		return target, nil, errors.Trace(err)
	}
	matchedVersion, matchedTools := filtered.Newest()

	return matchedVersion, matchedTools, nil
}

func (ctx UpgradeContext) Resolve(target version.Number) (version.Number, error) {
	if target == version.Zero {
		resolved, err := ctx.ResolveTarget(target)
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else {
		matched, tools, err := ctx.MatchTarget(target)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	if context.chosen == context.agent {
		return errUpToDate
	}
}

type OutputErr struct {
	Output  string
	Message string
}

func (err OutputErr) Error() string {
	return Message
}

func (ctx UpgradeContext) CheckTarget(target version.Number) error {
	if target == version.Zero {
		return nil
	}

	if target.Major > ctx.Agent.Major {
		// We can only upgrade a major version if we're currently on 1.25.2 or later
		// and we're going to 2.0.x, and the version was explicitly requested.
		if ctx.Agent.Major != 1 {
			return errors.NewNotValid(nil, "cannot upgrade to version incompatible with CLI")
		}

		var output []string
		if target.Major > 2 || target.Minor > 0 {
			output = append(output, "Upgrades to %s must first go through juju 2.0.")
		}
		if ctx.Agent.Minor < 25 || (ctx.Agent.Minor == 25 && ctx.Agent.Patch < 2) {
			output = append(output, "Upgrades to juju 2.0 must first go through juju 1.25.2 or higher.")
		}
		if len(output) > 0 {
			// TODO(ericsnow) Use errors.NewNotValid().
			return &OutputErr{
				Output:  strings.Join(output, "\n"),
				Message: "cannot upgrade to version incompatible with CLI",
			}
		}
	} else if target.Major < ctx.Agent.Major {
		return errors.NewNotValid(nil, "cannot upgrade to version incompatible with CLI")
	}

	//////////////////////////

	if ctx.Agent.Major == 1 {
		// If the client is on a version before 1.20, they must upgrade
		// through 1.20.14.
		if ctx.Agent.Minor < 20 {
			if target.Major > 1 || (target.Major == 1 && target.Minor > 20) {
				return errors.NewNotValid(nil, "unsupported upgrade\n\nEnvironment must first be upgraded to 1.20.14.\n    juju upgrade-juju --version=1.20.14")
			}
		}
	}

	if target.Major == 1 {
		if ctx.Agent.Major == 1 {
			// Skip 1.21 and 1.23 if the agents are not already on them.
			if ctx.Agent.Minor != target.Minor && (target.Minor == 21 || target.Minor == 23) {
				return errors.NewNotValid(nil, fmt.Sprintf("unsupported upgrade\n\nUpgrading to %s is not supported. Please upgrade to the latest 1.25 release.", target))
			}
		}
	}

	// Disallow major.minor version downgrades.
	if target.Major < ctx.Agent.Major || (target.Major == ctx.Agent.Major && target.Minor < ctx.Agent.Minor) {
		// TODO(fwereade): I'm a bit concerned about old agent/CLI tools even
		// *connecting* to environments with higher agent-versions; but ofc they
		// have to connect in order to discover they shouldn't. However, once
		// any of our tools detect an incompatible version, they should act to
		// minimize damage: the CLI should abort politely, and the agents should
		// run an Upgrader but no other tasks.
		return errors.NewNotValid(nil, fmt.Sprintf("cannot change version from %s to %s", ctx.Agent, target))
	}

	return nil
}

type envClient interface {
	EnvironmentGet() (map[string]interface{}, error)
}

func getEnvVersion(client envClient) (version.Number, error) {
	var agentVersion version.Number
	if client == nil {
		return agentVersion, nil
	}

	attrs, err := client.EnvironmentGet()
	if err != nil {
		return agentVersion, errors.Trace(err)
	}
	cfg, err := config.New(config.NoDefaults, attrs)
	if err != nil {
		return agentVersion, errors.Trace(err)
	}
	agentVersion, ok := cfg.AgentVersion()
	if !ok {
		// Can't happen. In theory.
		return agentVersion, errors.Errorf("incomplete environment configuration")
	}
	return agentVersion, nil
}

//type toolsClient interface {
//	FindTools(majorVersion, minorVersion int, series, arch string) (params.FindToolsResult, error)
//	UploadTools(r io.Reader, vers version.Binary, additionalSeries ...string) (*coretools.Tools, error)
//}
//
//func listTools(client toolsClient, major int) (tools.List, error) {
//	findResult, err := client.FindTools(major, -1, "", "")
//	if err != nil {
//		return nil, err
//	}
//
//	err = findResult.Error
//	if params.IsCodeNotFound(err) {
//		return errors.NewNotFound(err, fmt.Sprintf("tools for %s.X.X not found", major))
//	}
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//
//	return findResult.List, nil
//}
//
//func uploadTools(client toolsClient, target version.Number) (version.Number, error) {
//	//target = decideUploadVersion(target, tools)
//
//	builtTools, err := sync.BuildToolsTarball(&target, "upgrade")
//	if err != nil {
//		return nil, err
//	}
//	defer os.RemoveAll(builtTools.Dir)
//
//	var uploaded *coretools.Tools
//	toolsPath := path.Join(builtTools.Dir, builtTools.StorageName)
//	logger.Infof("uploading tools %v (%dkB) to Juju state server", builtTools.Version, (builtTools.Size+512)/1024)
//	f, err := os.Open(toolsPath)
//	if err != nil {
//		return nil, err
//	}
//	defer f.Close()
//
//	additionalSeries := version.OSSupportedSeries(builtTools.Version.OS)
//	uploaded, err = context.apiClient.UploadTools(f, builtTools.Version, additionalSeries...)
//	if err != nil {
//		return nil, err
//	}
//
//	return uploaded, nil
//}
//
//// decideUploadVersion returns a copy of the supplied version with a build number
//// higher than any of the available tools that share its major, minor, and patch.
//func decideUploadVersion(target version.Number, allAvailable []version.Number) version.Number {
//	decided := target // a copy
//
//	decided.Build++
//	for _, available := range allAvailable {
//		if available.Major != decided.Major || available.Minor != decided.Minor || available.Patch != decided.Patch {
//			continue
//		}
//
//		if decided.Build < available.Build {
//			decided.Build = available.Build + 1
//		}
//	}
//	return decided
//}
