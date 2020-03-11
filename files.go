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
	assetsPath  string
	assetsTheme string
	assets      map[string][]byte
)

// ClearAsset clears the asset's cache
func ClearAsset() (err error) {
	if _, err = os.Stat(assetsPath); os.IsNotExist(err) {
		return nil
	}
	if err == nil {
		assets = make(map[string][]byte)
		err = filepath.Walk(assetsPath, func(path string, info os.FileInfo, err error) error {
			var data []byte
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			data, err = ioutil.ReadFile(path)
			assets[filepath.ToSlash(path[len(assetsPath)+1:])] = data
			return err
		})
	}
	return
}

// LoadCustomAsset sets assets folder and load resources
func LoadCustomAsset(dir, theme string) error {
	assetsTheme = theme
	assetsPath = dir
	return ClearAsset()
}

// FileAsset return the file data
func FileAsset(fname string) (data []byte) {
	var ok bool

	if data, ok = assets[fname]; ok {
		return
	}
	data, _ = FSByte(false, path.Join(`/eonza-assets`, fname))
	return
}

// WebAsset return the file data
func WebAsset(fname string) (data []byte) {
	return FileAsset(path.Join(`themes`, assetsTheme, fname))
}

// TemplateAsset returns the template of the web-page
func TemplateAsset(fname string) []byte {
	return WebAsset(path.Join(`templates`, fname+`.tpl`))
}

// LanguageAsset returns the language resources
func LanguageAsset(lng string) []byte {
	return FileAsset(path.Join(`languages`, lng+`.yaml`))
}
