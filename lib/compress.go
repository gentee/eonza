// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package lib

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GzipCompress(input []byte) ([]byte, error) {
	var (
		buf bytes.Buffer
		err error
	)
	zw := gzip.NewWriter(&buf)
	zw.Name = "data"
	zw.Comment = ""
	zw.ModTime = time.Now()
	_, err = zw.Write(input)
	if err != nil {
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GzipDecompress(input []byte) (out []byte, err error) {
	var (
		gz *gzip.Reader
	)
	gz, err = gzip.NewReader(bytes.NewBuffer(input))
	if err != nil {
		return
	}
	defer gz.Close()
	out, err = io.ReadAll(gz)
	return
}

func unpackFile(finfo os.FileInfo, reader io.Reader, dest string) error {
	if finfo.IsDir() {
		return os.MkdirAll(dest, finfo.Mode())
	}
	target, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, finfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		target.Close()
		os.Chtimes(dest, finfo.ModTime(), finfo.ModTime())
	}()
	_, err = io.Copy(target, reader)
	return err
}

func UnpackTar(reader io.Reader, dir string) error {
	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := header.Name
		path := dir
		folder := filepath.Dir(strings.TrimRight(name, `/`))
		if len(folder) > 0 {
			path = filepath.Join(path, folder)
		}
		path = filepath.Join(path, header.FileInfo().Name())
		switch header.Typeflag {
		case tar.TypeDir, tar.TypeReg:
			if err = unpackFile(header.FileInfo(), tr, path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("UnpackTar: uknown type: %d in %s", header.Typeflag, header.Name)
		}
	}
	return nil
}
