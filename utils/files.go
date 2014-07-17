// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package utils

import (
	"fmt"
	"io"
	"os"
)

var openFile = os.Open

type File interface {
	io.ReadCloser
	Stat() (os.FileInfo, error)
}

func FileSizeCheck(file File) (size int64, err error) {
	stat, err := file.Stat()
	if err != nil {
		err = fmt.Errorf("could not get file info: %v", err)
	} else {
		size = stat.Size()
		if size == 0 {
			err = fmt.Errorf("file is empty")
		} else if size < 0 {
			err = fmt.Errorf("negative file size: %d", size)
		}
	}
	return
}

func OpenFileCheck(filename string) (io.ReadCloser, error) {
	file, err := openFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	_, err = FileSizeCheck(file)
	if err != nil {
		return nil, err
	}
	return file, nil
}
