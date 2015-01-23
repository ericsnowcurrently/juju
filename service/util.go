// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"strings"
)

func hasPrefix(name string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func contains(strList []string, str string) bool {
	for _, contained := range strList {
		if str == contained {
			return true
		}
	}
	return false
}

// fromSlash is borrowed from cloudinit/renderers.go.
func fromSlash(path string, initSystem string) string {
	// TODO(ericsnow) Is this right?
	// If initSystem is "" then just do the default.

	// TODO(ericsnow) Just use filepath.FromSlash for local?

	if initSystem == InitSystemWindows {
		return strings.Replace(path, "/", `\`, -1)
	}
	return path
}

type fileOp struct {
	kind   string
	target string
	mode   os.FileMode
}

type cachedFileOperations struct {
	files map[string]bytes.Buffer
	dirs  []string
	ops   []fileOp
}

func newCachedFileOperations() *cachedFileOperations {
	return &cachedFileOperations{
		files: make(map[string]bytes.Buffer),
	}
}

func (cfo *cachedFileOperations) exists(name string) (bool, error) {
	cfo.ops = append(cfo.ops, fileOp{
		kind:   "exists",
		target: name,
	})

	if _, ok := cfo.files[name]; ok {
		return true, nil
	}
	if contains(cfo.dirs, name) {
		return true, nil
	}
	return false, nil
}

func (cfo *cachedFileOperations) mkdirs(dirname string, mode os.FileMode) error {
	cfo.ops = append(cfo.ops, fileOp{
		kind:   "mkdirs",
		target: dirname,
		mode:   mode,
	})

	if !contains(cfo.dirs, dirname) {
		cfo.dirs = append(cfo.dirs, dirname)
	}
	return nil
}

func (cfo *cachedFileOperations) readfile(filename string) ([]byte, error) {
	cfo.ops = append(cfo.ops, fileOp{
		kind:   "readfile",
		target: filename,
	})

	data, ok := cfo.files[filename]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data.Bytes(), nil
}

func (cfo *cachedFileOperations) create(filename string) (io.WriteCloser, error) {
	cfo.ops = append(cfo.ops, fileOp{
		kind:   "createfile",
		target: filename,
	})

	if _, ok := cfo.files[filename]; ok {
		return nil, os.ErrExist
	}
	if contains(cfo.dirs, dirname) {
		return nil, os.ErrExist
	}
	cfo.files[filename] = new(bytes.Buffer)
	return ioutil.NopCloser(cfo.files[filename])
}

func (cfo *cachedFileOperations) remove(name string) error {
	cfo.ops = append(cfo.ops, fileOp{
		kind:   "removeall",
		target: name,
	})

	if _, ok := cfo.files; ok {
		delete(cfo.files, name)
	} else {
		for i, dirname := range cfo.dirs {
			if name == dirname {
				cfo.dirs = append(cfo.dirs[:i], cfo.dirs[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (cachedFileOperations) chmod(name string, mode os.FileMode) error {
	cfo.ops = append(cfo.ops, fileOp{
		kind:   "chmod",
		target: name,
		mode:   mode,
	})

	if _, ok := cfo.files[name]; ok {
		return nil
	}
	if contains(cfo.dirs, dirname) {
		return nil
	}
	return os.ErrNotExist
}
