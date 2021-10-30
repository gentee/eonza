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
	Name     string
	Dir      bool
	Data     []byte
	Original []byte
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
			Dir:  hdr.Typeflag == tar.TypeDir,
			Name: hdr.Name,
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

// Restore restores original data.
func (tfs *TarFS) Restore() {
	for name, f := range tfs.Files {
		if f.Original != nil {
			tfs.Files[name].Data = f.Original
			tfs.Files[name].Original = nil
		}
	}
}

// Redefine redefines asset data.
func (tfs *TarFS) Redefine(name string, data []byte) {
	if f, ok := tfs.Files[name]; ok {
		if f.Original == nil {
			f.Original = make([]byte, len(f.Data))
			copy(f.Original, f.Data)
		}
		f.Data = data
	}
}
