// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdpath "path"
	"path/filepath"

	"github.com/juju/juju/environs"
	stdstorage "github.com/juju/juju/environs/storage"
	"github.com/juju/juju/state"
)

const storageBase = "/backups"

var notFound = errors.New("not found")

// StorageName returns the path in environment storage where a backup
// should be stored.
func StorageName(filename string) string {
	return resolveName(filepath.Base(filename))
}

func resolveName(name string) string {
	if name == "" {
		return storageBase
	}
	// Use of path.Join instead of filepath.Join is intentional - this
	// is an environment storage path not a filesystem path.
	return stdpath.Join(storageBase, name)
}

func info(storage stdstorage.Storage, name, path string) (*BackupInfo, error) {
	file, err := storage.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error reading backup info (%s): %v", name, err)
	}
	defer file.Close()

	var info BackupInfo
	err = json.NewDecoder(file).Decode(&info)
	if err != nil {
		return nil, fmt.Errorf("error decoding backup info (%s): %v", name, err)
	}

	if info.Name != name {
		return nil, fmt.Errorf("failed to get backup info for %q", name)
	}

	return &info, nil
}

// BackupStorage is the API for the storage used by backups.  Aside from
// extra methods, BackupStorage differs from storage.Storage in that
// List() takes a regex pattern on which to match rather than just a
// prefix.  Direct use of Put() is discouraged.
type BackupStorage interface {
	Info(name string) (*BackupInfo, error)
	Archive(name string) (io.ReadCloser, error)
	Add(info *BackupInfo, archive io.Reader) error
	// In common with storage.StorageReader:
	List(pattern string) ([]string, error)
	URL(name string) (string, error)
	// In common with storage.StorageWriter:
	Remove(name string) error
	RemoveAll() error
	CheckConsistency(cleanup bool) error
}

//---------------------------
// backup storage implementation

// XXX Store info in state rather than raw storage?

// Storage provides an API for storing backup info and archives.
type storage struct {
	st  *state.State
	raw stdstorage.Storage
}

// NewBackupStorage returns a new backup storage based on the state.
func NewBackupStorage(
	st *state.State, stor stdstorage.Storage,
) (BackupStorage, error) {
	var err error
	if stor == nil {
		stor, err = environs.GetStorage(st)
		if err != nil {
			return nil, fmt.Errorf("error getting environment storage: %v", err)
		}
	}

	backupStorage := storage{
		st:  st,
		raw: stor,
	}
	return &backupStorage, nil
}

func (s *storage) resolveInfo(name string) string {
	base := resolveName(name)
	return stdpath.Join(base, "info")
}

func (s *storage) resolveArchive(name string) string {
	base := resolveName(name)
	return stdpath.Join(base, "archive")
}

func (s *storage) Info(name string) (*BackupInfo, error) {
	path := s.resolveInfo(name)
	return info(s.raw, name, path)
}

func (s *storage) Archive(name string) (io.ReadCloser, error) {
	path := s.resolveArchive(name)
	return s.raw.Get(path)
}

func (s *storage) stAdd(info *BackupInfo, name string) error {
	if name == "" {
		name = info.Name
	}
	// XXX Finish!
	return nil
}

func (s *storage) setInfo(info *BackupInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("error encoding backup info (%s): %v", info.Name, err)
	}
	file := bytes.NewBuffer(data)

	path := s.resolveInfo(info.Name)
	return s.raw.Put(path, file, int64(len(data)))
}

func (s *storage) Add(info *BackupInfo, archive io.Reader) error {
	// XXX Check for existing?
	err := s.stAdd(info, "")
	if err != nil {
		return err
	}
	err = s.setInfo(info)
	if err != nil {
		return err
	}
	path := s.resolveArchive(info.Name)
	return s.raw.Put(path, archive, info.Size)
}

func (s *storage) stList(pattern string) ([]string, error) {
	// XXX Finish!
	return nil, notFound
}

func (s *storage) List(pattern string) ([]string, error) {
	// Try s.st first.
	names, err := s.stList(pattern)
	if err == nil {
		return names, nil
	} else if err != notFound {
		return nil, err
	}

	// Fall back to raw storage.
	path := resolveName(pattern)
	return s.raw.List(path)
}

func (s *storage) URL(name string) (string, error) {
	path := s.resolveArchive(name)
	return s.raw.URL(path)
}

func (s *storage) stRemove(name string) error {
	// XXX Finish!
	return nil
}

func (s *storage) Remove(name string) error {
	err := s.stRemove(name)
	if err != nil {
		return err
	}
	path := resolveName(name)
	return s.raw.Remove(path)
}

func (s *storage) RemoveAll() error {
	names, err := s.List("")
	if err != nil {
		return err
	}
	for _, name := range names {
		err := s.Remove(name)
		if err != nil {
			return err
		}
	}
	return nil
}

type storageInconsistency struct {
	State    []string
	Storage  []string
	Info     []string
	Archives []string
}

func (e *storageInconsistency) Error() string {
	msg := "unmatched names (state: %v) (storage: %v) (info: %v) (archives: %v)"
	return fmt.Sprintf(msg, e.State, e.Storage, e.Info, e.Archives)
}

func (s *storage) CheckConsistency(cleanup bool) error {
	st, stor, err := s.compareNames(cleanup)
	if err != nil {
		return err
	}
	info, archives, err := s.checkStorage(cleanup)
	if err != nil {
		return err
	}
	if len(st) > 0 || len(stor) > 0 || len(info) > 0 || len(archives) > 0 {
		return &storageInconsistency{st, stor, info, archives}
	}
	return nil
}

func (s *storage) compareNames(cleanup bool) ([]string, []string, error) {
	// Pull the state names.
	stNames, err := s.stList("")
	if err == notFound {
		// State isn't set up for backups so we're good to go.
		return []string{}, []string{}, nil
	} else if err != nil {
		return nil, nil, err
	}

	// Pull the storage names.
	path := resolveName("")
	names, err := s.raw.List(path)
	if err != nil {
		return nil, nil, err
	}

	// Compare them.
	for i := len(stNames); i > 0 && len(names) > 0; i-- {
		stName := stNames[i]
		for j, name := range names {
			if stName == name {
				names[j] = names[0]
				names = names[1:]
				stNames[i] = stNames[0]
				stNames = stNames[1:]
				break
			}
		}
	}
	if cleanup {
		for _, name := range stNames {
			s.stRemove(name)
		}
		for _, name := range names {
			s.stAdd(nil, name)
		}
	}
	return stNames, names, nil
}

func (s *storage) checkStorage(cleanup bool) ([]string, []string, error) {
	infos := make([]string, 0)
	archives := make([]string, 0)
	names, err := s.raw.List(resolveName(""))
	if err != nil {
		return nil, nil, err
	}
	for _, name := range names {
		info, err := s.raw.List(s.resolveInfo(name))
		if err != nil {
			return nil, nil, err
		}
		if len(info) == 0 {
			infos = append(infos, name)
		}
		archive, err := s.raw.List(s.resolveArchive(name))
		if err != nil {
			return nil, nil, err
		}
		if len(archive) == 0 {
			archives = append(archives, name)
		}
	}
	return infos, archives, nil
}
