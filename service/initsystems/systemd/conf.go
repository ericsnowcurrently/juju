// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/coreos/go-systemd/unit"
	"github.com/juju/errors"

	"github.com/juju/juju/service/initsystems"
)

var limitMap = map[string]string{
	"as":         "LimitAS",
	"core":       "LimitCORE",
	"cpu":        "LimitCPU",
	"data":       "LimitDATA",
	"fsize":      "LimitFSIZE",
	"memlock":    "LimitMEMLOCK",
	"msgqueue":   "LimitMSGQUEUE",
	"nice":       "LimitNICE",
	"nofile":     "LimitNOFILE",
	"nproc":      "LimitNPROC",
	"rss":        "LimitRSS",
	"rtprio":     "LimitRTPRIO",
	"sigpending": "LimitSIGPENDING",
	"stack":      "LimitSTACK",
}

// Validate returns an error if the service is not adequately defined.
func Validate(name string, conf initsystems.Conf) error {
	err := conf.Validate(name)
	if err != nil {
		return errors.Trace(err)
	}

	if conf.Out != "" && conf.Out != "syslog" {
		return errors.NotValidf("conf.Out value %q (Options are syslog)", conf.Out)
	}

	for k := range conf.Limit {
		if _, ok := limitMap[k]; !ok {
			return errors.NotValidf("conf.Limit key %q", k)
		}
	}
	return nil
}

// Serialize serializes the provided Conf for the named service. The
// resulting data will be in the prefered format for consumption by
// the init system.
func Serialize(name string, conf initsystems.Conf) ([]byte, error) {
	if err := Validate(name, conf); err != nil {
		return nil, errors.Trace(err)
	}

	var unitOptions []*unit.UnitOption
	unitOptions = append(unitOptions, &unit.UnitOption{
		Section: "Unit",
		Name:    "Description",
		Value:   conf.Desc,
	})
	unitOptions = append(unitOptions, &unit.UnitOption{
		Section: "Service",
		Name:    "ExecStart",
		Value:   conf.Cmd,
	})
	if conf.Out != "" {
		unitOptions = append(unitOptions, &unit.UnitOption{
			Section: "Service",
			Name:    "StandardOutput",
			Value:   conf.Out,
		})
		unitOptions = append(unitOptions, &unit.UnitOption{
			Section: "Service",
			Name:    "StandardError",
			Value:   conf.Out,
		})
	}
	for k, v := range conf.Env {
		unitOptions = append(unitOptions, &unit.UnitOption{
			Section: "Service",
			Name:    "Environment",
			Value:   fmt.Sprintf(`"%q=%q"`, k, v),
		})
	}
	for k, v := range conf.Limit {
		unitOptions = append(unitOptions, &unit.UnitOption{
			Section: "Service",
			Name:    limitMap[k],
			Value:   v,
		})
	}

	data, err := ioutil.ReadAll(unit.Serialize(unitOptions))
	return data, errors.Trace(err)
}

// Deserialize parses the provided data (in the init system's prefered
// format) and populates a new Conf with the result.
func Deserialize(data []byte, name string) (*initsystems.Conf, error) {
	opts, err := unit.Deserialize(bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.Trace(err)
	}

	var conf initsystems.Conf

	for _, uo := range opts {
		switch uo.Section {
		case "Service":
			switch {
			case uo.Name == "ExecStart":
				conf.Cmd = uo.Value
			case uo.Name == "StandardError", uo.Name == "StandardOutput":
				// TODO(wwitzel3) We serialize Standard(Error|Output)
				// to the same thing, but we should probably make sure they match
				conf.Out = uo.Value
			case uo.Name == "Environment":
				if conf.Env == nil {
					conf.Env = make(map[string]string)
				}
				var value = uo.Value
				if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
					value = value[1 : len(value)-1]
				}
				parts := strings.SplitN(value, "=", 2)
				if len(parts) != 2 {
					return nil, errors.NotValidf("service environment value %q", uo.Value)
				}
				conf.Env[parts[0]] = parts[1]
			case strings.HasPrefix(uo.Name, "Limit"):
				if conf.Limit == nil {
					conf.Limit = make(map[string]string)
				}
				for k, v := range limitMap {
					if v == uo.Name {
						conf.Limit[k] = v
						break
					}
				}
			default:
				return nil, errors.NotSupportedf("service directive %q")
			}

		case "Unit":
			switch uo.Name {
			case "Description":
				conf.Desc = uo.Value
			default:
				return nil, errors.NotSupportedf("unit directive %q")
			}
		default:
			return nil, errors.NotSupportedf("section %q")
		}
	}

	if name == "" {
		name = "<>"
	}
	err = Validate(name, conf)
	return &conf, errors.Trace(err)
}
