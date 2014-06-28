// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/state/backup"
	"github.com/juju/utils"
)

type ClientErrorCode string

const (
	CodeRequestNotCreated ClientErrorCode = "could not create RPC request"
	CodeRequestNotSent    ClientErrorCode = "could not send RPC request"
	CodeRequest
)

type RawHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type RawRPCHandler func(data io.Reader, statusCode int) error

func (c *Client) GetRawHTTPClient() RawHTTPClient {
	httpclient := utils.GetValidatingHTTPClient()
	tlsconfig := tls.Config{RootCAs: c.st.certPool, ServerName: "anything"}
	httpclient.Transport = utils.NewHttpTLSTransport(&tlsconfig)
	return httpclient
}

// XXX Add args and body parameters.
func (c *Client) sendRawRPC(method string, errResponse interface{}) (io.ReadCloser, error) {
	// Prepare the request.
	url := fmt.Sprintf("%s/%s", c.st.serverRoot, method)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %v", err)
	}
	req.SetBasicAuth(c.st.tag, c.st.password)

	// Send the request.
	httpclient := c.GetRawHTTPClient()
	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send HTTP request: %v", err)
	}
	//defer resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		// API server is 1.16 or older, so charm upload
		// is not supported; notify the client.
		return nil, &params.Error{
			Message: "charm upload is not supported by the API server",
			Code:    params.CodeNotImplemented,
		}
	}
	// Handle a bad response code.
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read error data: %v", err)
		}

		if err := json.Unmarshal(body, errResponse); err != nil {
			return nil, fmt.Errorf("could not unpack error data: %v", err)
		}
		return nil, fmt.Errorf("request failed on server")
	}
	// XXX Call a handler here instead of returning the data.
	return resp, nil
}

// backup

// Backup requests a state-server backup file from the server and saves it to
// the local filesystem. It returns the name of the file created.
// The backup can take a long time to prepare and be a large file, depending
// on the system being backed up.
func (c *Client) Backup(backupFilePath string) (string, error) {
	if backupFilePath == "" {
		formattedDate := time.Now().Format(backup.TimestampFormat)
		backupFilePath = fmt.Sprintf(backup.FilenameTemplate, formattedDate)
	}

	/*
		// Prepare the request.
		rpcmethod := "backup"
		url := fmt.Sprintf("%s/%s", c.st.serverRoot, rpcmethod)
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			return "", fmt.Errorf("cannot create backup request: %v", err)
		}
		req.SetBasicAuth(c.st.tag, c.st.password)

		// Send the request.
		httpclient := c.GetRawHTTPClient()
		resp, err := httpclient.Do(req)
		if err != nil {
			return "", fmt.Errorf("cannot fetch backup: %v", err)
		}
		defer resp.Body.Close()

		// Handle a bad response code.
		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("cannot read backup response: %v", err)
			}

			var jsonResponse params.BackupResponse
			if err := json.Unmarshal(body, &jsonResponse); err != nil {
				return "", fmt.Errorf("cannot unmarshal backup response: %v", err)
			}

			return "", fmt.Errorf("error fetching backup: %v", jsonResponse.Error)
		}
	*/
	var errorResponse params.BackupResponse
	resp, err := c.sendRawRPC("backup", &errorResponse)
	if err != nil {
		if errorResponse.Error != "" {
			err = fmt.Errorf("%v: %v", err, errorResponse.Error)
		}
		return "", err
	}
	defer resp.Body.Close()

	// Write out the archive.
	err = c.writeBackupFile(backupFilePath, resp.Body)
	if err != nil {
		return "", err
	}

	// Validate the result.
	err = c.validateBackupHash(backupFilePath, resp)
	if err != nil {
		return backupFilePath, err
	}

	return backupFilePath, nil
}

func (c *Client) writeBackupFile(backupFilePath string, body io.Reader) error {
	file, err := os.Create(backupFilePath)
	if err != nil {
		return fmt.Errorf("Error creating backup file: %v", err)
	}
	defer file.Close()
	_, err = io.Copy(file, body)
	if err != nil {
		return fmt.Errorf("Error writing the backup file: %v", err)
	}
	return nil
}

func (c *Client) validateBackupHash(backupFilePath string, resp *http.Response) error {
	// Get the expected hash.
	// XXXX what to do if the digest header is missing?
	// just log and return? (Seems hostile to delete it.)
	digest := resp.Header.Get("Digest")
	if digest == "" || digest[:4] != "SHA=" {
		logger.Warningf("SHA digest missing from response. Can't verify the backup file.")
		return nil
	}
	expected := digest[4:]

	// Get the actual hash.
	tarball, err := os.Open(backupFilePath)
	if err != nil {
		return fmt.Errorf("could not open backup file: %s", backupFilePath)
	}
	defer tarball.Close()

	actual, err := backup.GetHash(tarball)
	if err != nil {
		return err
	}

	// Compare the hashes.
	if actual != expected {
		return fmt.Errorf("archive hash did not match value from server: %s != %s",
			actual, expected)
	}
	return nil
}

/*
func (c *Client) Backup(backupFilePath string) (string, error) {
	if backupFilePath == "" {
		formattedDate := time.Now().Format(backup.TimestampFormat)
		backupFilePath = fmt.Sprintf(backup.FilenameTemplate, formattedDate)
	}

	// Prepare the upload request.
	req, err := c.getBackupRequest()
	if err != nil {
		return "", err
	}
	httpclient, err := c.getBackupRawClient()
	if err != nil {
		return "", err
	}

	// Send the request.
	resp, err := c.sendBackupRequest(httpclient, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Handle the response.
	data, err := c.handleBackupResponse(resp)
	if err != nil {
		return "", err
	}

	// Write out the archive.
	err = c.writeBackupFile(backupFilePath, data)
	if err != nil {
		return "", err
	}

	// Validate the result.
	err = c.validateBackupHash(backupFilePath, resp)
	if err != nil {
		return "", err
	}

	return backupFilePath, nil
}

func (c *Client) getBackupRequest() (*http.Request, error) {
	url := fmt.Sprintf("%s/backup", c.st.serverRoot)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create backup request: %v", err)
	}
	req.SetBasicAuth(c.st.tag, c.st.password)
	return req, nil
}

func (c *Client) getBackupRawClient() (*http.Client, error) {
	//rawclient := utils.GetNonValidatingHTTPClient()
	rawclient := utils.GetValidatingHTTPClient()
	tlsconfig := tls.Config{RootCAs: c.st.certPool,
		ServerName: "anything"}
	//return &http.Client{Transport: &transport}, nil
	rawclient.Transport = utils.NewHttpTLSTransport(&tlsconfig)
	return rawclient, nil
}

func (c *Client) sendBackupRequest(rawclient *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := rawclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch backup: %v", err)
	}
	return resp, nil
}

func (c *Client) handleBackupResponse(resp *http.Response) (io.Reader, error) {
	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}

	// Handle a bad response code.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read backup response: %v", err)
	}

	var jsonResponse params.BackupResponse
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return nil, fmt.Errorf("cannot unmarshal backup response: %v", err)
	}

	return nil, fmt.Errorf("error fetching backup: %v", jsonResponse.Error)
}
*/
