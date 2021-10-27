// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"path"

	"eonza/internal/tarfs"

	"github.com/kataras/golog"
)

//go:embed assets/assets.tar.gz
var webTar []byte

var (
	WebFS *tarfs.TarFS
)

func LoadAssets(run bool) {
	var err error
	WebFS, err = tarfs.NewTarFS(webTar)
	if err != nil {
		golog.Fatal(err)
	}
}

// WebAsset return the file data
func WebAsset(fname string) (data []byte) {
	return WebFS.File(path.Join(`themes`, assetsTheme, fname))
}
