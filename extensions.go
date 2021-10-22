// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	es "eonza/script"
	"fmt"
)

type ExtensionInfo struct {
	Title    string `json:"title" yaml:"title"`
	Desc     string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Help     string `json:"help,omitempty" yaml:"help,omitempty"`
	HelpLang string `json:"helplang,omitempty" yaml:"helplang,omitempty"`
}

type ExtensionReview struct {
	ExtensionInfo
}

type Extensions struct {
	Exts []ExtensionReview `json:"exts" yaml:"exts"`
}

type Extension struct {
	ExtensionInfo
	Langs  map[string]map[string]string `json:"langs,omitempty" yaml:"langs,omitempty"`
	Params []es.ScriptParam             `json:"params,omitempty" yaml:"params,omitempty"`
}

type ExtSettings struct {
	Disable bool
	Values  map[string]interface{}
}

func LoadExtensions() {
	fmt.Println(`Load`)
}

func InstallExtension(name string) {
	fmt.Println(`Install`, name)
}

func UninstallExtension(name string) {
	fmt.Println(`Uninstall`, name)
}

func EnableExtension(name string) {
	fmt.Println(`Enable`, name)
}

func DisableExtension(name string) {
	fmt.Println(`Disable`, name)
}
