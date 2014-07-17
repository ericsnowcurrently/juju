// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package archives

import (
	"bytes"
	"compress/gzip"
)

func GzipData(data string) (string, error) {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write([]byte(data))
	gz.Close()
	if err != nil {
		return "", err
	}
	return compressed.String(), nil
}
