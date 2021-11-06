// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	es "eonza/script"
	"fmt"
	"net/http"
	"sort"

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
		ext.Installed = false
		Assets.Packages[name] = ext
	}
}

func InstallPackage(name string) {
	if cfg.playground {
		// TODO: error
	}
	fmt.Println(`Install`, name)
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

func findPackage(name string) (*Package, error) {
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
	if err = CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	if ext, err = findPackage(c.Param("name")); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, &ExtResponse{
		Params: ext.Params,
	})
}
