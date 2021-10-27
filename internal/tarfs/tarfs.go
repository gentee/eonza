// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package tarfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
)

type FileFS struct {
	Data []byte
}

type TarFS struct {
	Files map[string]*FileFS
}

// NewTarFS decompresses input tar.gz data.
func NewTarFS(data []byte) (*TarFS, error) {
	var (
		tfs TarFS
		buf bytes.Buffer
	)

	zr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(&buf, zr); err != nil {
		return nil, err
	}
	if err := zr.Close(); err != nil {
		return nil, err
	}
	tfs.Files = make(map[string]*FileFS, 16)
	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		ifile := FileFS{
			Data: make([]byte, hdr.Size),
		}
		if _, err = tr.Read(ifile.Data); err != nil && err != io.EOF {
			return nil, err
		}
		tfs.Files[hdr.Name] = &ifile
	}
	return &tfs, nil
}

// File returns the content of the specified file.
func (tfs *TarFS) File(name string) []byte {
	if f, ok := tfs.Files[name]; ok {
		return f.Data
	}
	return []byte{}
}
