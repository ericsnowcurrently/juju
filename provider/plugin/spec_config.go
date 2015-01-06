// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package plugin

import (
	"github.com/juju/errors"
	"github.com/juju/schema"

	"github.com/juju/juju/environs/config"
)

// TODO(ericsnow) Use charm.optionTypeCheckers instead.
var fieldTypeCheckers = map[string]schema.Checker{
	"string":  schema.String(),
	"int":     schema.Int(),
	"float":   schema.Float(),
	"boolean": schema.Bool(),
}

type ConfigField struct {
	Type         string      `yaml:"type"`
	Description  string      `yaml:"description,omitempty"`
	OSEnvVarName string      `yaml:"env_var,omitempty"`
	Default      interface{} `yaml:"default,omitempty"`
	Secret       bool        `yaml:"is_secret,omitempty"`
	Immutable    bool        `yaml:"is_immutable,omitempty"`
}

// error replaces any supplied non-nil error with a new error describing a
// validation failure for the supplied value.
func (cf ConfigField) error(err *error, name string, value interface{}) {
	if *err != nil {
		*err = fmt.Errorf("field %q expected %s, got %#v", name, cf.Type, value)
	}
}

// coerce returns an appropriately-typed value for the supplied value, or
// returns an error if it cannot be converted to the correct type. Nil values
// are always considered valid.
func (cf ConfigField) coerce(name string, value interface{}) (_ interface{}, err error) {
	if value == nil {
		return nil, nil
	}
	defer cf.error(&err, name, value)

	checker, ok := fieldTypeCheckers[field.Type]
	if !ok {
		// TODO(ericsnow) Do not panic!
		panic(fmt.Errorf("field %q has unknown type %q", name, cf.Type))
	}
	return checker.Coerce(value, nil)
}

// parse returns an appropriately-typed value for the supplied string, or
// returns an error if it cannot be parsed to the correct type.
func (cf ConfigField) parse(name, str string) (_ interface{}, err error) {
	defer cf.error(&err, name, str)

	switch option.Type {
	case "string":
		return str, nil
	case "int":
		return strconv.ParseInt(str, 10, 64)
	case "float":
		return strconv.ParseFloat(str, 64)
	case "boolean":
		return strconv.ParseBool(str)
	}
	// TODO(ericsnow) Do not panic!
	panic(fmt.Errorf("option %q has unknown type %q", name, option.Type))
}

// ConfigSpec represents the supported configuration fields for a
// provider as declared in its config.yaml file.
type ConfigSpec struct {
	Fields map[string]ConfigField `yaml:"fields"`
}

// ReadConfigSpec reads a ConfigSpec in YAML format.
func ReadConfigSpec(r io.Reader) (*ConfigSpec, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var spec *ConfigSpec
	if err := goyaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	if spec == nil {
		return nil, errors.New("invalid config spec: empty configuration")
	}

	for name, field := range spec.Fields {
		switch field.Type {
		case "string", "int", "float", "boolean":
		case "":
			// Default to string.
			field.Type = "string"
		default:
			return nil, errors.Errorf("invalid config spec: field %q has unknown type %q", name, option.Type)
		}
		def := field.Default
		if def == "" && option.Type == "string" {
			continue
		}
		def, err = field.coerce(name, def)
		if err != nil {
			field.error(&err, name, def)
			return nil, errors.Annotate(err, "invalid field default")
		}
		field.Default = def
	}
	return spec, nil
}

func (cs *ConfigSpec) boilerplate() string {
	// TODO(ericsnow) Build from fields.
}

func (cs *configSpec) fields() (map[string]schema.Checker, error) {
	result := make(map[string]schema.Checker, len(cs.raw))
	for _, field := range cs.raw {
		if field.typeName == "" {
			result[field.name] = schema.String
			continue
		}

		checker, ok := fieldTypes[field.typeName]
		if !ok {
			return nil, errors.Errorf("unknown field type %q", field.typeName)
		}
		result[field.name] = checker
	}
	return result, nil
}

func (cs *configSpec) defaults() map[string]interface{} {
	result := make(map[string]interface{})
	for _, field := range cs.raw {
		if field.defaultValue == nil {
			continue
		}
		result[field.name] = field.defaultValue
	}
	return result, nil
}

func (cs *configSpec) osEnvVars() map[string]string {
	result := make(map[string]interface{})
	for _, field := range cs.raw {
		if field.defaultValue == nil {
			continue
		}
		result[field.name] = field.defaultValue
	}
	return result, nil
}

func (cs *configSpec) secret() []string {
	result := make(map[string]interface{})
	for _, field := range cs.raw {
		if field.defaultValue == nil {
			continue
		}
		result[field.name] = field.defaultValue
	}
	return result, nil
}

func (cs *configSpec) immutable() []string {
	result := make(map[string]interface{})
	for _, field := range cs.raw {
		if field.defaultValue == nil {
			continue
		}
		result[field.name] = field.defaultValue
	}
	return result, nil
}

const (
	// These are not GCE-official environment variable names.
	osEnvPrivateKey    = "GCE_PRIVATE_KEY"
	osEnvClientID      = "GCE_CLIENT_ID"
	osEnvClientEmail   = "GCE_CLIENT_EMAIL"
	osEnvRegion        = "GCE_REGION"
	osEnvProjectID     = "GCE_PROJECT_ID"
	osEnvImageEndpoint = "GCE_IMAGE_URL"

	cfgPrivateKey    = "private-key"
	cfgClientID      = "client-id"
	cfgClientEmail   = "client-email"
	cfgRegion        = "region"
	cfgProjectID     = "project-id"
	cfgImageEndpoint = "image-endpoint"
)

// boilerplateConfig will be shown in help output, so please keep it up to
// date when you change environment configuration below.
var boilerplateConfig = `
gce:
  type: gce

  # Google Auth Info
  private-key: 
  client-email:
  client-id:

  # Google instance info
  # region: us-central1
  project-id:
  # image-endpoint: https://www.googleapis.com
`[1:]

var osEnvFields = map[string]string{
	osEnvPrivateKey:    cfgPrivateKey,
	osEnvClientID:      cfgClientID,
	osEnvClientEmail:   cfgClientEmail,
	osEnvRegion:        cfgRegion,
	osEnvProjectID:     cfgProjectID,
	osEnvImageEndpoint: cfgImageEndpoint,
}

var configFields = schema.Fields{
	cfgPrivateKey:    schema.String(),
	cfgClientID:      schema.String(),
	cfgClientEmail:   schema.String(),
	cfgRegion:        schema.String(),
	cfgProjectID:     schema.String(),
	cfgImageEndpoint: schema.String(),
}

var configDefaults = schema.Defaults{
	// See http://cloud-images.ubuntu.com/releases/streams/v1/com.ubuntu.cloud:released:gce.json
	cfgImageEndpoint: "https://www.googleapis.com",
}

var configSecretFields = []string{
	cfgPrivateKey,
}

var configImmutableFields = []string{
	cfgPrivateKey,
	cfgClientID,
	cfgClientEmail,
	cfgRegion,
	cfgProjectID,
	cfgImageEndpoint,
}

type environConfig struct {
	*config.Config
	attrs map[string]interface{}
}

/*
func (c *environConfig) auth() gceAuth {
	return gceAuth{
		clientID:    c.attrs[cfgClientID].(string),
		clientEmail: c.attrs[cfgClientEmail].(string),
		privateKey:  []byte(c.attrs[cfgPrivateKey].(string)),
	}
}

func (c *environConfig) newConnection() *gceConnection {
	return &gceConnection{
		region:    c.attrs[cfgRegion].(string),
		projectID: c.attrs[cfgProjectID].(string),
	}
}
*/

func validateConfig(cfg, old *config.Config) (map[string]interface{}, error) {
	// Check for valid changes and coerce the values (base config first
	// then custom).
	if err := config.Validate(cfg, old); err != nil {
		return nil, errors.Trace(err)
	}
	validated, err := cfg.ValidateUnknownAttrs(configFields, configDefaults)
	return validated, errors.Trace(err)
}

func checkNotChanged(cfg, old *config.Config) error {
	if old == nil {
		return nil
	}

	attrs := old.UnknownAttrs()
	for _, field := range configImmutableFields {
		if attrs[field] != validated[field] {
			return nil, errors.Errorf("%s: cannot change from %v to %v", field, attrs[field], validated[field])
		}
	}
	return nil
}

/*
	// TODO(ericsnow) Follow up with someone on if it is appropriate
	// to call Apply here.
	cfg, err = ecfg.Config.Apply(ecfg.attrs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ecfg.Config = cfg
	return ecfg, nil
*/

func handleInvalidField(err error) error {
	vErr := err.(*config.InvalidConfigValue)
	if vErr.Reason == nil && vErr.Value == "" {
		key := osEnvFields[vErr.Key]
		return errors.Errorf("%s: must not be empty", key)
	}
	return err
}
