// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package lib

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
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
	out, err = ioutil.ReadAll(gz)
	return
}
