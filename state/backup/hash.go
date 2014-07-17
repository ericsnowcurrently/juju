// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

// We could move these to juju/utils/hash

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
)

// GetHash returns the SHA1 hash generated from the provided file.
func GetHash(file io.Reader) (string, error) {
	shahash := sha1.New()
	_, err := io.Copy(shahash, file)
	if err != nil {
		return "", fmt.Errorf("could not extract hash: %v", err)
	}

	return base64.StdEncoding.EncodeToString(shahash.Sum(nil)), nil
}

// GetHash returns the SHA1 hash generated from the provided compressed file.
func UncompressAndGetHash(compressed io.Reader, mimetype string) (string, error) {
	var archive io.ReadCloser
	var err error

	switch mimetype {
	case "application/x-tar":
		return "", fmt.Errorf("not compressed: %s", mimetype)
	default:
		// Fall back to "application/x-tar-gz".
		archive, err = gzip.NewReader(compressed)
	}
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}
	defer archive.Close()

	return GetHash(archive)
}

// GetHashByFilename opens the file, unpacks it if compressed, and
// computes the SHA1 hash of the contents.
func GetHashDefault(filename string) (string, error) {
	file, err := OpenFileCheck(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return UncompressAndGetHash(file, CompressionType)
}

// GetHashByFilename opens the file, unpacks it if compressed, and
// computes the SHA1 hash of the contents.
func GetHashByFilename(filename string) (string, error) {
	file, err := OpenFileCheck(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mimetype string
	if strings.HasSuffix(filename, ".tar.gz") {
		mimetype = "application/x-tar-gz"
	} else {
		ext := filepath.Ext(filename)
		mimetype = mime.TypeByExtension(ext)
		if mimetype == "" {
			return "", fmt.Errorf("unsupported filename (%s)", filename)
		}
	}

	if mimetype == "application/x-tar" {
		return GetHash(file)
	} else {
		return UncompressAndGetHash(file, mimetype)
	}
}
