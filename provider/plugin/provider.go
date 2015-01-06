// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package plugin

import (
	"github.com/juju/errors"

	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
)

const (
	pluginsDir = "" // TODO(ericsnow) finish!
)

func init() {
	// TODO(ericsnow) Add to providers/all.

	// We ignore any failures here. Usually it won't matter. When it
	// does juju will simply fail for not finding the provider.
	// TODO(ericsnow) Is this okay?
	dirnames, err := findPlugins(pluginsDir)
	if err != nil {
		logger.Errorf("error while looking for providers: %v", err)
		return
	}

	for _, dirname := range dirnames {
		name := filepath.Base(dirname)
		plugin, err := newPlugin(name, dirname)
		if err != nil {
			// TODO(ericsnow) Okay to ignore bad providers here?
			logger.Errorf("failed to init provider at %q", dirname)
			continue
		}
		environs.RegisterProvider(plugin.name, plugin)
	}
}

func findPlugins(pluginsDir string) ([]string, error) {
	var dirnames []string

	infos, err := ioutil.ReadDir(pluginsDir)
	if os.IsNotExist(err) {
		logger.Errorf("no providers found")
		return dirnames, nil
	}
	if err != nil {
		return dirnames, errors.Trace(err)
	}

	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		dirnames = append(dirnames, filepath.Join(pluginsDir, info.Name()))
	}
	return dirnames, nil
}

type plugin struct {
	name  string
	hooks providerHooks
	spec  providerSpec
}

func newPlugin(name, dirname string) (*plugin, error) {
	spec, err := ReadSpec(dirname)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err := spec.CheckHooks(); err != nil {
		return errors.Trace(err)
	}

	provider := plugin{
		name: name,
		spec: spec,
	}
	provider.hooks = providerHooks{&provider}
	return &provider, nil
}

func (p plugin) Open(cfg *config.Config) (environs.Environ, error) {
	env := &environ{
		name:     cfg.Name(),
		provider: &p,
	}
	env.hooks = envHooks{env}

	if err := env.SetConfig(cfg); err != nil {
		return nil, errors.Trace(err)
	}
	return env, nil
}

func (p plugin) Prepare(ctx environs.BootstrapContext, cfg *config.Config) (environs.Environ, error) {
	env, err := p.Open(cfg)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if ctx.ShouldVerifyCredentials() {
		if err := env.(*environ).hooks.verifyCredentials(); err != nil {
			return nil, err
		}
	}
	return env, nil
}

func (p plugin) validate(cfg, old *config.Config) (map[string]interface{}, error) {
	valid, err := validateConfig(cfg, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = p.hooks.checkConfig(valid)
	return valid, errors.Trace(err)
}

func (p plugin) updateFromOSEnv(cfg *config.Config) (*config.Config, error) {
	updates, err := p.spec.config.osEnvUpdates(cfg)
	if err != nil {
		return nil, errors.Annotate(err, "invalid OS env vars")
	}
	cfg, err = cfg.Apply(updates)
	if err != nil {
		return nil, errors.Annotate(err, "invalid OS env vars")
	}
	return cfg, nil
}

func (p plugin) Validate(cfg, old *config.Config) (*config.Config, error) {
	cfg, err := p.updateFromOSEnv(cfg)
	if err != nil {
		return nil, errors.Trace(err)
	}

	valid, err := p.validate(cfg, nil)
	if err != nil {
		return nil, errors.Annotate(err, "invalid config")
	}
	if old != nil {
		oldValid, err := p.validate(old, nil)
		if err != nil {
			return nil, errors.Annotate(err, "invalid base config")
		}
		old, err = old.Apply(oldValid)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if valid, err = p.validate(cfg, old); err != nil {
			return nil, errors.Annotate(err, "invalid config change")
		}
		if err := p.spec.config.checkNotChanged(valid, old); err != nil {
			return nil, errors.Annotate(err, "invalid config change")
		}
	}
	cfg, err = cfg.Apply(valid)
	return cfg, errors.Trace(err)
}

func (p plugin) SecretAttrs(cfg *config.Config) (map[string]string, error) {
	cfg, err := p.updateFromOSEnv(cfg)
	if err != nil {
		return nil, errors.Trace(err)
	}

	validated, err := validateConfig(cfg, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	secretAttrs := make(map[string]string, len(p.spec.config.secret()))
	for _, key := range p.spec.config.secret() {
		secretAttrs[key] = valid[key].(string)
	}
	return secretAttrs, nil
}

func (plugin) BoilerplateConfig() string {
	return p.spec.config.boilerplate()
}

// CheckPlugin is a helper for provider authors to verify that their
// hooks are acceptable. It corresponds to the like-named juju subcommand.
func CheckPlugin(dirname string) error {
	if _, err := ReadSpec(dirname); err != nil {
		return errors.Trace(err)
	}
	if err := checkHooks(dirname); err != nil {
		return errors.Trace(err)
	}
	return nil
}
