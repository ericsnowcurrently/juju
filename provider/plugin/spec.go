// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package plugin

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Spec encapsulates access to data and operations
// on a provider directory.
type Spec struct {
	Dirname string

	meta     *Meta
	config   *ConfigSpec
	revision int
}

// ReadSpec returns a Spec representing an expanded provider directory.
func ReadSpec(dirname string) (*Spec, error) {
	file, err := os.Open(filepath.Join(dirname, "metadata.yaml"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	meta, err = ReadMeta(file)
	file.Close()
	if err != nil {
		return nil, errors.Trace(err)
	}

	file, err = os.Open(filepath.Join(dirname, "config.yaml"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	config, err = ReadConfigSpec(file)
	file.Close()
	if err != nil {
		return nil, errors.Trace(err)
	}

	var revision int
	file, err := os.Open(filepath.Join(dirname, "revision"))
	if err != nil {
		logger.Errorf("could not open revision file: %v", err)
		// Leave the revision at 0.
	} else {
		_, err = fmt.Fscan(file, &revision)
		file.Close()
		if err != nil {
			return nil, errors.New("invalid revision file")
		}
	}

	spec := Spec{
		Dirname:  dirname,
		meta:     meta,
		config:   config,
		revision: revision,
	}
	return &spec, nil
}

func (spec *Spec) storeRevision(revision int) error {
	filename := filepath.Join(spec.Dirname, "revision")
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Trace(err)
	}
	defer file.Close()

	_, err = file.Write([]byte(strconv.Itoa(revision)))
	return errors.Trace(err)
}
