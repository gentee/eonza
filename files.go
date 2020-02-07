// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
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
	assetsBox = filepath.Join(string(filepath.Separator)+`eonza-assets`, theme)
	dir = filepath.Join(dir, theme)
	if _, err := os.Stat(dir); err == nil {
		isAssets = true
	}
	assetsDir = dir
	fmt.Println(`Asset`, isAssets, dir)
	return nil
}

// FileAsset return the file data
func FileAsset(fname string) (data []byte) {
	var (
		ok bool
	)
	if data, ok = assets[fname]; !ok {
		if isAssets {
			filePath := filepath.Join(assetsDir, filepath.FromSlash(fname))
			fmt.Println(`Path`, filePath)
			if _, err := os.Stat(filePath); err == nil {
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

// TemplateAsset returns the template of the web-page
func TemplateAsset(fname string) []byte {
	return FileAsset(filepath.Join(`templates`, fname+`.tpl`))
}
