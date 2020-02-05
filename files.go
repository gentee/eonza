// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

var (
	assetsTheme string
	assetsDir   string
	isAssets    bool
	assetsBox   string
	assets      = make(map[string][]byte)
)

// SetAsset sets assets folder
func SetAsset(dir, theme string) error {
	assetsBox = `\eonza-assets\` + theme
	dir = filepath.Join(dir, theme)
	if _, err := os.Stat(dir); os.IsExist(err) {
		isAssets = true
	}
	assetsDir = dir
	return nil
}

// FileAsset return the file data
func FileAsset(fname string) (data []byte) {
	var (
		ok bool
	)
	if data, ok = assets[fname]; !ok {
		if isAssets {
			filePath := filepath.Join(assetsDir, fname)
			if _, err := os.Stat(filePath); os.IsExist(err) {
				data, _ = ioutil.ReadFile(filePath)
				assets[fname] = data
				return
			}
		}
		assets[fname] = []byte{}
	}
	if len(data) > 0 {
		return
	}
	data, _ = FSByte(false, path.Join(assetsBox, fname))
	return
}
