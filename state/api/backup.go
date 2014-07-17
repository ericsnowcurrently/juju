// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/juju/juju/state/backup"
)

var getBackupHash = backup.GetHashDefault

// Backup requests a state-server backup file from the server and saves it to
// the local filesystem. It returns the name of the file created.
// The backup can take a long time to prepare and be a large file, depending
// on the system being backed up.
func (c *Client) Backup(filename string, validate bool) (string, string, error) {
	if filename == "" {
		filename = backup.DefaultFilename()
	}

	// Send the request and copy the backup into the file.
	filename, digest, err := c.doBackup(filename)
	if err != nil {
		return filename, "", err
	}

	// Validate the result.
	if validate {
		err = validateBackupHash(filename, digest)
		if err != nil {
			return filename, digest, err
		}
	}

	return filename, digest, nil
}

func (c *Client) doBackup(filename string) (string, string, error) {
	// Open the backup file.
	file, err := os.Create(filename)
	if err != nil {
		return "", "", fmt.Errorf("error creating backup file: %v", err)
	}
	defer file.Close()

	// Send the request.
	resp, err := c.sendHTTPRequest(&backup.APIBinding)
	if err != nil {
		return filename, "", err
	}
	defer resp.Body.Close()

	// Write out the archive.
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		err := fmt.Errorf("error writing the backup file: %v", err)
		return filename, "", err
	}

	// Extract the digest from the header.
	digest, err := extractBackupDigestFromHeader(resp)
	if err != nil {
		err := fmt.Errorf("could not extract SHA-1 digest from HTTP header: %v", err)
		return filename, "", err
	}

	return filename, digest, nil
}

func extractBackupDigestFromHeader(resp *http.Response) (string, error) {
	digests, err := backup.ParseDigestHeader(resp.Header)
	if err != nil {
		return "", err
	}
	digest, exists := digests[backup.DigestAlgorithm]
	if !exists {
		return "", fmt.Errorf("'SHA' digest missing from response")
	}
	return digest, nil
}

func validateBackupHash(backupFilePath, expected string) error {
	// Get the actual hash.
	actual, err := getBackupHash(backupFilePath)
	if err != nil {
		return fmt.Errorf("could not verify backup file: %v", err)
	}

	// Compare the hashes.
	if actual != expected {
		return fmt.Errorf("archive hash did not match value from server: %s != %s", actual, expected)
	}
	return nil
}
