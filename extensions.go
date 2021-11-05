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

type ExtensionInfo struct {
	Title    string `json:"title" yaml:"title"`
	Desc     string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Help     string `json:"help,omitempty" yaml:"help,omitempty"`
	HelpLang string `json:"helplang,omitempty" yaml:"helplang,omitempty"`
	// Calculated fields
	Installed bool `json:"installed"`
}

type ExtensionReview struct {
	ExtensionInfo
	Name string `json:"name"`
}

type ExtensionsResponse struct {
	List  []ExtensionReview `json:"list,omitempty"`
	Error string            `json:"error,omitempty"`
}

type Extension struct {
	ExtensionInfo `yaml:"info"`
	Version       string                       `yaml:"version"`
	Langs         map[string]map[string]string `json:"langs,omitempty" yaml:"langs,omitempty"`
	Params        []es.ScriptParam             `json:"params,omitempty" yaml:"params,omitempty"`
}

type ExtSettings struct {
	Values map[string]interface{}
}

func LoadExtensions() {
	// TODO: Check installed extensions
	for name, ext := range Assets.Extensions {
		ext.Installed = false
		Assets.Extensions[name] = ext
	}
}

func InstallExtension(name string) {
	if cfg.playground {
		// TODO: error
	}
	fmt.Println(`Install`, name)
}

func UninstallExtension(name string) {
	if cfg.playground {
		// TODO: error
	}
	fmt.Println(`Uninstall`, name)
}

func ExtensionsList(c echo.Context) *ExtensionsResponse {
	lang := c.(*Auth).Lang
	glob := &langRes[GetLangId(c.(*Auth).User)]
	ret := make([]ExtensionReview, 0)
	for name, ext := range Assets.Extensions {
		ext.Title = es.ReplaceVars(ext.Title, ext.Langs[lang], glob)
		ext.Desc = es.ReplaceVars(ext.Desc, ext.Langs[lang], glob)
		ret = append(ret, ExtensionReview{
			ExtensionInfo: ext.ExtensionInfo,
			Name:          name,
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Installed == ret[j].Installed {
			return ret[i].Title < ret[j].Title
		}
		return ret[i].Installed
	})
	return &ExtensionsResponse{
		List: ret,
	}
}

func extsHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, ExtensionsList(c))
}
