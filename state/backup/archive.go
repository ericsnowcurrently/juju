// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

// We could move these to juju/utils/archives

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func writeArchive(fileList []string, outfile io.Writer, strip string, compress bool) error {
	var err error

	if compress {
		gzw := gzip.NewWriter(outfile)
		defer gzw.Close()
		outfile = gzw
	}
	tarw := tar.NewWriter(outfile)
	defer tarw.Close()

	for _, ent := range fileList {
		if err = AddToTarfile(ent, strip, tarw); err != nil {
			return fmt.Errorf("backup failed: %v", err)
		}
	}

	return nil
}

// CreateArchive returns a sha1 hash of targetPath after writing out the
// archive.  This archive holds the files listed in fileList. If
// compress is true, the archive will also be gzip compressed.
func CreateArchive(fileList []string, targetPath, strip string, compress bool) (string, error) {
	// Create the archive file.
	tarball, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("cannot create backup file %q", targetPath)
	}
	defer tarball.Close()

	// Write out the archive.
	err = writeArchive(fileList, tarball, strip, compress)
	if err != nil {
		return "", err
	}

	// Return the hash
	tarball.Seek(0, os.SEEK_SET)
	var data []byte
	tarball.Read(data)
	tarball.Seek(0, os.SEEK_SET)
	if compress {
		// We want the hash of the uncompressed file, since the
		// compressed archive will have a different hash depending on
		// the compression format.
		//	    return GetHash(tarball)
		return UncompressAndGetHash(tarball, CompressionType)
	} else {
		return GetHash(tarball)
	}
}

// AddToTarfile creates an entry for the given file
// or directory in the given tar archive.
func AddToTarfile(fileName, strip string, tarw *tar.Writer) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	fInfo, err := f.Stat()
	if err != nil {
		return err
	}
	h, err := tar.FileInfoHeader(fInfo, "")
	if err != nil {
		return fmt.Errorf("cannot create tar header for %q: %v", fileName, err)
	}
	h.Name = filepath.ToSlash(strings.TrimPrefix(fileName, strip))
	if err := tarw.WriteHeader(h); err != nil {
		return fmt.Errorf("cannot write header for %q: %v", fileName, err)
	}
	if !fInfo.IsDir() {
		if _, err := io.Copy(tarw, f); err != nil {
			return fmt.Errorf("failed to write %q: %v", fileName, err)
		}
		return nil
	}
	if !strings.HasSuffix(fileName, string(os.PathSeparator)) {
		fileName = fileName + string(os.PathSeparator)
	}

	for {
		names, err := f.Readdirnames(100)
		if len(names) == 0 && err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error reading directory %q: %v", fileName, err)
		}
		for _, name := range names {
			if err := AddToTarfile(filepath.Join(fileName, name), strip, tarw); err != nil {
				return err
			}
		}
	}

}

func FileSizeCheck(file *os.File) (size int64, err error) {
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

func OpenFileCheck(filename string) (*os.File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	_, err = FileSizeCheck(file)
	if err != nil {
		return nil, err
	}
	return file, nil
}
