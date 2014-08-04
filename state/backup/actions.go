// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

import ()

type Action string

const (
	ActionNoop   Action = "no-op"
	ActionCreate Action = "create"
)

type apiClient interface {
	Backup(args BackupArgs) (BackupResult, error)
}

type BackupAPI interface {
	Create(name string) (info *backup.BackupInfo, url string, err error)
}

func HandleRequest(api BackupAPI, args *BackupArgs) (*BackupResult, error) {
	var result BackupResult

	switch args.Action {
	case ActionNoop:
		// Do nothing.
	case ActionCreate:
		info, url, err := api.Create(args.Name)
		if err != nil {
			return nil, err
		}
		result.Info = *info
		result.URL = url
	default:
		return nil, fmt.Errorf("unsupported backup action: %q", args.Action)
	}

	return &result, nil
}

//-----------------------------------------

var (
	createBackup = backup.CreateBackup
)

type backupServerAPI struct {
	dbinfo  *backup.DBConnInfo
	storage backup.BackupStorage
}

func NewBackupServerAPI(
	dbinfo *backup.DBConnInfo, stor backup.BackupStorage,
) BackupAPI {
	return newBackupServerAPI(dbinfo, stor)
}

var newBackupServerAPI = func(
	dbinfo *backup.DBConnInfo, stor backup.BackupStorage,
) BackupAPI {
	api := backupServerAPI{
		dbinfo:  dbinfo,
		storage: stor,
	}
	return &api
}

func (ba *backupServerAPI) Create(name string) (*backup.BackupInfo, string, error) {
	info, err := createBackup(ba.dbinfo, ba.storage, name, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error creating backup: %v", err)
	}
	// We have to wait, so we might as well include the URL.
	URL, err := ba.storage.URL(info.Name)
	if err != nil {
		return info, "", nil
	}
	return info, URL, nil
}

//-----------------------------------------

type backupClientAPI struct {
	client apiClient
}

func NewBackupClientAPI(client apiClient) BackupAPI {
	return newBackupClientAPI(client)
}

func newBackupClientAPI(client apiClient) BackupAPI {
	api := backupClientAPI{
		client: client,
	}
	return &api
}

func (ba *backupClientAPI) Create(name string) (*backup.BackupInfo, string, error) {
	args := BackupArgs{
		Action: ActionCreate,
		Name:   name,
	}
	result, err := ba.client.Backup(args)
	if err != nil {
		return nil, "", err
	}
	return &result.Info, result.URL, nil
}
