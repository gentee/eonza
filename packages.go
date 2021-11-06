// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	es "eonza/script"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

type PackageInfo struct {
	Title    string `json:"title" yaml:"title"`
	Desc     string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Help     string `json:"help,omitempty" yaml:"help,omitempty"`
	HelpLang string `json:"helplang,omitempty" yaml:"helplang,omitempty"`
	// Calculated fields
	Installed bool `json:"installed"`
}

type PackageReview struct {
	PackageInfo
	Name string `json:"name"`
}

type PackagesResponse struct {
	List  []PackageReview `json:"list,omitempty"`
	Error string          `json:"error,omitempty"`
}

type ExtResponse struct {
	Params []es.ScriptParam  `json:"params,omitempty"`
	Values map[string]string `json:"values,omitempty"`
	Error  string            `json:"error,omitempty"`
}

type Package struct {
	PackageInfo `yaml:"info"`
	Version     string                       `yaml:"version"`
	Langs       map[string]map[string]string `json:"langs,omitempty" yaml:"langs,omitempty"`
	Params      []es.ScriptParam             `json:"params,omitempty" yaml:"params,omitempty"`
}

type ExtSettings struct {
	Values map[string]interface{}
}

func LoadPackages() {
	// TODO: Check installed packages
	for name, ext := range Assets.Packages {
		if _, err := os.Stat(filepath.Join(cfg.PackagesDir, name)); err == nil {
			ext.Installed = true
			Assets.Packages[name] = ext
		}
	}
}

func UninstallPackage(name string) {
	if cfg.playground {
		// TODO: error
	}
	fmt.Println(`Uninstall`, name)
}

func PackagesList(c echo.Context) *PackagesResponse {
	lang := c.(*Auth).Lang
	glob := &langRes[GetLangId(c.(*Auth).User)]
	ret := make([]PackageReview, 0)
	for name, ext := range Assets.Packages {
		ext.Title = es.ReplaceVars(ext.Title, ext.Langs[lang], glob)
		ext.Desc = es.ReplaceVars(ext.Desc, ext.Langs[lang], glob)
		ret = append(ret, PackageReview{
			PackageInfo: ext.PackageInfo,
			Name:        name,
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Installed == ret[j].Installed {
			return ret[i].Title < ret[j].Title
		}
		return ret[i].Installed
	})
	return &PackagesResponse{
		List: ret,
	}
}

func packagesHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, PackagesList(c))
}

func findPackage(c echo.Context, name string) (*Package, error) {
	if err := CheckAdmin(c); err != nil {
		return nil, err
	}
	v, ok := Assets.Packages[name]
	if !ok {
		return nil, fmt.Errorf(`Cannot find %s package`, name)
	}
	return &v, nil
}

func packageHandle(c echo.Context) error {
	var (
		err error
		ext *Package
	)
	if ext, err = findPackage(c, c.Param("name")); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, &ExtResponse{
		Params: ext.Params,
	})
}

func packageInstallHandle(c echo.Context) error {
	var (
		err error
		ext *Package
	)
	if cfg.playground {
		return jsonError(c, fmt.Errorf(`Access denied`))
	}
	name := c.Param("name")
	if ext, err = findPackage(c, name); err != nil {
		return jsonError(c, err)
	}
	if name == `tests` {
		storage.Events[`test`] = &Event{
			ID:        lib.RndNum(),
			Name:      `test`,
			Script:    `data-print`,
			Token:     `TEST_TOKEN`,
			Whitelist: `::1/128, 127.0.0.0/31`,
			Active:    true,
		}
		if err = SaveStorage(); err != nil {
			return jsonError(c, err)
		}
	}
	path := filepath.Join(cfg.PackagesDir, name)
	if err := os.MkdirAll(path, 0777); err != nil {
		return jsonError(c, err)
	}
	for _, f := range PackagesFS.List {
		if strings.HasPrefix(f.Name, name+`/files`) {
			fullName := filepath.Join(cfg.PackagesDir, f.Name)
			if f.Dir {
				if err := os.MkdirAll(fullName, 0777); err != nil {
					os.RemoveAll(path)
					return jsonError(c, err)
				}
			} else if err = os.WriteFile(fullName, f.Data, 0666); err != nil {
				os.RemoveAll(path)
				return jsonError(c, err)
			}
		}
	}
	ext.Installed = true
	Assets.Packages[name] = *ext
	return c.JSON(http.StatusOK, PackagesList(c))
}

func packageUninstallHandle(c echo.Context) error {
	var (
		err error
		ext *Package
	)
	if cfg.playground {
		return jsonError(c, fmt.Errorf(`Access denied`))
	}
	name := c.Param("name")
	if ext, err = findPackage(c, name); err != nil {
		return jsonError(c, err)
	}
	// TODO: проверка на зависимости для каждого скрипта пакета
	path := filepath.Join(cfg.PackagesDir, name)
	if err := os.RemoveAll(path); err != nil {
		return jsonError(c, err)
	}
	ext.Installed = false
	Assets.Packages[name] = *ext
	if name == `tests` {
		delete(storage.Events, `test`)
		SaveStorage()
	}
	return c.JSON(http.StatusOK, PackagesList(c))
}
