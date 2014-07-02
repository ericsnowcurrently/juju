// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/juju/errors"
	ziputil "github.com/juju/utils/zip"

	"github.com/juju/juju/environs"
	"github.com/juju/juju/state/api/params"
)

//---------------------------
// request processing

// ExtractFile extracts raw file data from an HTTP response and writes
// it to a local file.
func ExtractFile(r *http.Request, filename string, mimetype string) (string, error) {
	if mimetype != "" {
		contentType := r.Header.Get("Content-Type")
		if contentType != mimetype {
			return "", fmt.Errorf("expected Content-Type: %v, got: %v", mimetype, contentType)
		}
	}

	// Prepare the output file.
	var outfile *os.File
	if filename != "" {
		outfile, err = os.Create(filename)
		if err != nil {
			return "", fmt.Errorf("could not create file (%s): %v", filename, err)
		}
	} else {
		outfile, err := ioutil.TempFile("", "")
		if err != nil {
			return "", fmt.Errorf("could not create temp file: %v", err)
		}
	}
	defer outfile.Close()

	// Store the data in the output file.
	if _, err := io.Copy(outfile, r.Body); err != nil {
		os.Remove(outfile.Name())
		return "", fmt.Errorf("could not extract file upload: %v", err)
	}

	return outfile.Name(), nil
}

//---------------------------
// response helpers

// SendJSONRaw sends an HTTP response with the result serialized to JSON.
// If force is true, an error during serialization is propagated out.
// Otherwise the function always returns nil.
func SendJSONRaw(w http.ResponseWriter, statusCode int, result interface{}, force bool) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	body, err := json.Marshal(err)
	if err != nil && force {
		return err
	}
	w.Write(body)
	return nil
}

// SendJSON sends an HTTP response with the result serialized to JSON.
// The HTTP status code is set to StatusOK (200).
func SendJSON(w http.ResponseWriter, result interface{}) {
	force := false
	SendJSONRaw(w, http.StatusOK, result, force)
}

// SendErrorRaw sends an HTTP response with the error serialized to JSON.
// If force is true, an error during serialization is propagated out.
// Otherwise the function always returns nil.
func SendErrorRaw(w http.ResponseWriter, statusCode int, err *params.Error, force bool) error {
	return SendJSONRaw(w, statusCode, err, force)
}

func SendErrorRawString(w http.ResponseWriter, statusCode int, msg string, force bool) error {
	err := params.Error{Message: msg}
	return SendErrorRaw(w, statusCode, err, force)
}

// SendBadRequest sends an HTTP response with the error wrapped in
// params.Error and serialized to JSON.  The HTTP status code is set to
// BadRequest (400).  This indicates that something was wrong with the
// request itself.
func SendBadRequest(w http.ResponseWriter, err error) {
	SendBadRequestString(w, err.Error())
}

func SendBadRequestString(w http.ResponseWriter, msg string) {
	force := false
	SendErrorRawString(w, http.StatusBadRequest, msg, force)
}

// SendError sends an HTTP response with the error wrapped in
// params.Error and serialized to JSON.  It is used when something went
// wrong on the server side while handling the API method, but which is
// not an actual failure of the API method.  The HTTP status code is set
// to InternalServerError (500).
func SendError(w http.ResponseWriter, err error) {
	SendErrorString(w, err.Error())
}

func SendErrorString(w http.ResponseWriter, msg string) {
	apiError := params.Error{Message: msg}
	force := true
	SendErrorRaw(w, http.StatusInternalServerError, &apiError, force)
}

// SendFailure sends an HTTP response with the error wrapped in
// params.Error and serialized to JSON.  It is used when the server-side
// encounters an actual failure of the API method.  The HTTP status code
// is set to InternalServerError (500).
func SendFailure(w http.ResponseWriter, failure *params.Error) {
	force := true
	SendErrorRaw(w, http.StatusInternalServerError, &failure, force)
}

// SendBinary sends an HTTP response with the file written to the response.
func SendBinary(w http.ResponseWriter, file io.Reader, size uint, mimetype string) {
	// Send the header.
	if mimetype == "" {
		mimetype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mimetype)
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	w.WriteHeader(http.StatusOK)

	io.Copy(w, file)
}

func SendFile(w http.ResponseWriter, file *os.File, mimetype string) {
	var size int
	info, err := file.Stat()
	if err != nil {
		size = info.Size()
	}
	SendBinary(w, file, size, mimetype)
}

/*
	// TODO(fwereade) 2014-01-27 bug #1285685
	// This doesn't handle symlinks helpfully, and should be talking in
	// terms of bundles rather than zip readers; but this demands thought
	// and design and is not amenable to a quick fix.
	zipReader, err := zip.OpenReader(bundle.Path)
	if err != nil {
		http.Error(
			w, fmt.Sprintf("unable to read charm: %v", err),
			http.StatusInternalServerError)
		return
	}
	defer zipReader.Close()
	for _, file := range zipReader.File {
		if path.Clean(file.Name) != filePath {
			continue
		}
		fileInfo := file.FileInfo()
		if fileInfo.IsDir() {
			http.Error(w, "directory listing not allowed", http.StatusForbidden)
			return
		}
		contents, err := file.Open()
		if err != nil {
			http.Error(
				w, fmt.Sprintf("unable to read file %q: %v", filePath, err),
				http.StatusInternalServerError)
			return
		}
		defer contents.Close()
		ctype := mime.TypeByExtension(filepath.Ext(filePath))
		if ctype != "" {
			w.Header().Set("Content-Type", ctype)
		}
		w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, contents)
		return
	}
	http.NotFound(w, r)
	return
}
*/

func SendFileByName(w http.ResponseWriter, filename string, mimetype string) {
	if mimetype == "" {
		mimetype = mime.TypeByExtension(filepath.Ext(filePath))
	}

	infile := io.Open(filename)
	defer infile.Close()

	SendFile(w, infile, mimetype)
}
