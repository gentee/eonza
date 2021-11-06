// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"os"
	"path"
	"path/filepath"

	"eonza/internal/tarfs"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

//go:embed assets/assets.yaml
var assetsYaml []byte

//go:embed assets/web.tar.gz
var webTar []byte

//go:embed assets/stdlib.tar.gz
var stdlibTar []byte

//go:embed assets/init.tar.gz
var initTar []byte

//go:embed assets/packages.tar.gz
var packagesTar []byte

type CfgAssets struct {
	Templates []string           `yaml:"templates"`
	Languages []string           `yaml:"languages"`
	Packages  map[string]Package `yaml:"packages"`
}

var (
	customAssets string
	assetsTheme  string
	Assets       CfgAssets
	WebFS        *tarfs.TarFS
	StdlibFS     *tarfs.TarFS
	InitFS       *tarfs.TarFS
	PackagesFS   *tarfs.TarFS
)

func LoadAssets(run bool) {
	var err error
	if err = yaml.Unmarshal(assetsYaml, &Assets); err != nil {
		golog.Fatal(err)
	}
	WebFS, err = tarfs.NewTarFS(webTar)
	if err != nil {
		golog.Fatal(err)
	}
	if !run {
		StdlibFS, err = tarfs.NewTarFS(stdlibTar)
		if err != nil {
			golog.Fatal(err)
		}
		PackagesFS, err = tarfs.NewTarFS(packagesTar)
		if err != nil {
			golog.Fatal(err)
		}
	}
}

func InitAssets() {
	var err error
	InitFS, err = tarfs.NewTarFS(initTar)
	if err != nil {
		golog.Fatal(err)
	}
}

// WebAsset return the file data
func WebAsset(fname string) []byte {
	return WebFS.File(path.Join(`themes`, assetsTheme, fname))
}

// LanguageAsset returns the language resources
func LanguageAsset(lng string) []byte {
	return WebFS.File(path.Join(`languages`, lng+`.yaml`))
}

// TemplateAsset returns the template of the web-page
func TemplateAsset(fname string) []byte {
	return WebAsset(path.Join(`templates`, fname+`.tpl`))
}

// LoadCustomAsset sets assets folder and load resources
func LoadCustomAsset(dir, theme string) error {
	customAssets = dir
	assetsTheme = theme
	return RedefineAsset()
}

// RedefineAssets clears the asset's cache and load custom assets
func RedefineAsset() (err error) {
	if len(customAssets) == 0 {
		return
	}
	WebFS.Restore()
	if _, err = os.Stat(customAssets); os.IsNotExist(err) {
		return
	}
	if err == nil {
		err = filepath.Walk(customAssets, func(path string, info os.FileInfo, err error) error {
			var data []byte
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			data, err = os.ReadFile(path)
			WebFS.Redefine(filepath.ToSlash(path[len(customAssets)+1:]), data)
			return err
		})
	}
	return
}

func Asset(assetname, filename string) error {
	/*	data := FileAsset(assetname)
		if len(data) == 0 {
			return fmt.Errorf(`asset %s doesn't exist`, assetname)
		}
		return os.WriteFile(filename, data, 0666)*/
	return nil
}
