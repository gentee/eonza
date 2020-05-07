// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"eonza/lib"
	"fmt"
	"path"
	"strings"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

var (
	scripts map[string]*Script
)

type ParamType int

const (
	PCheckbox ParamType = iota
	PTextarea
)

type scriptSettings struct {
	Name  string `json:"name" yaml:"name"`
	Title string `json:"title" yaml:"title"`
	Desc  string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Unrun bool   `json:"unrun,omitempty" yaml:"unrun,omitempty"`
}

type scriptOptions struct {
	Initial  string `yaml:"initial,omitempty"`
	Default  string `yaml:"default,omitempty"`
	Required bool   `yaml:"required,omitempty"`
}

type scriptParam struct {
	Name    string    `json:"name" yaml:"name"`
	Title   string    `json:"title" yaml:"title"`
	Type    ParamType `json:"type" yaml:"type"`
	Options string    `json:"options,omitempty" yaml:"options,omitempty"`

	options scriptOptions
}

type scriptTree struct {
	Name     string                 `json:"name" yaml:"name"`
	Open     bool                   `json:"open,omitempty" yaml:"open,omitempty"`
	Disable  bool                   `json:"disable,omitempty" yaml:"disable,omitempty"`
	Values   map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	Children []scriptTree           `json:"children,omitempty" yaml:"children,omitempty"`
}

type Script struct {
	Settings scriptSettings `json:"settings" yaml:"settings"`
	Params   []scriptParam  `json:"params,omitempty" yaml:"params,omitempty"`
	Tree     []scriptTree   `json:"tree,omitempty" yaml:"tree,omitempty"`
	Code     string         `json:"code,omitempty" yaml:"code,omitempty"`
	folder   bool           // can have other commands inside
	embedded bool           // Embedded script
	initial  string         // Initial value
}

func getScript(name string) (script *Script) {
	return scripts[lib.IdName(name)]
}

func setScript(name string, script *Script) error {
	var ivalues map[string]string

	scripts[lib.IdName(name)] = script
	if len(script.Params) > 0 {
		ivalues = make(map[string]string)
	}
	for i, par := range script.Params {
		if len(par.Options) > 0 {
			var options scriptOptions
			if err := yaml.Unmarshal([]byte(par.Options), &options); err != nil {
				return err
			}
			script.Params[i].options = options
			if len(options.Initial) > 0 {
				ivalues[par.Name] = options.Initial
			}
		}
	}
	if len(ivalues) > 0 {
		initial, err := json.Marshal(ivalues)
		if err != nil {
			return err
		}
		script.initial = string(initial)
	}
	return nil
}

func delScript(name string) {
	name = lib.IdName(name)
	delete(scripts, name)
	delete(storage.Scripts, name)
}

func InitScripts() {
	scripts = make(map[string]*Script)
	isfolder := func(script *Script) bool {
		return script.Settings.Name == SourceCode ||
			strings.Contains(script.Code, `%body%`)
	}
	for _, tpl := range _escDirs["../eonza-assets/scripts"] {
		var script Script
		fname := tpl.Name()
		data := FileAsset(path.Join(`scripts`, fname))
		if err := yaml.Unmarshal(data, &script); err != nil {
			golog.Fatal(err)
		}
		script.embedded = true
		script.folder = isfolder(&script)
		if err := setScript(script.Settings.Name, &script); err != nil {
			golog.Fatal(err)
		}
	}
	for name, item := range storage.Scripts {
		if scripts[lib.IdName(name)] != nil {
			golog.Errorf(`The '%s' script has been loaded as embedded script`, name)
			continue
		}
		// TODO: this is a temporary fix
		if strings.Contains(name, `-`) {
			continue
		}
		//
		item.folder = isfolder(item)
		if err := setScript(name, item); err != nil {
			golog.Fatal(err)
		}
	}
}

func (script *Script) Validate() error {
	if !lib.ValidateSysName(script.Settings.Name) {
		return fmt.Errorf(Lang(`invalidfield`), Lang(`name`))
	}
	if len(script.Settings.Title) == 0 {
		return fmt.Errorf(Lang(`invalidfield`), Lang(`title`))
	}
	return nil
}

func scanTree(tree []scriptTree, name string) bool {
	for _, item := range tree {
		if item.Name == name {
			return true
		}
		if len(item.Children) > 0 && scanTree(item.Children, name) {
			return true
		}
	}
	return false
}

func ScriptDependences(name string) []ScriptItem {
	var ret []ScriptItem

	for _, item := range scripts {
		if item.embedded {
			continue
		}
		if scanTree(item.Tree, name) {
			ret = append(ret, ScriptToItem(item))
		}
	}
	return ret
}

func checkDep(name, title string) error {
	if deps := ScriptDependences(name); len(deps) > 0 {
		ret := make([]string, len(deps))
		for i, item := range deps {
			ret[i] = item.Title
		}
		return fmt.Errorf(Lang(`depscript`), title, strings.Join(ret, `,`))
	}
	return nil
}

func (script *Script) SaveScript(original string) error {
	if curScript := getScript(original); curScript != nil && curScript.embedded {
		return fmt.Errorf(Lang(`errembed`))
	}
	if len(original) > 0 && original != script.Settings.Name {
		if getScript(script.Settings.Name) != nil {
			return fmt.Errorf(Lang(`errscriptname`), script.Settings.Name)
		}
		if err := checkDep(original, script.Settings.Title); err != nil {
			return err
		}
		delScript(original)
	}
	script.folder = script.Settings.Name == SourceCode ||
		strings.Contains(script.Code, `%body%`)
	if err := setScript(script.Settings.Name, script); err != nil {
		return err
	}
	storage.Scripts[lib.IdName(script.Settings.Name)] = script
	return SaveStorage()
}

func DeleteScript(name string) error {
	script := getScript(name)
	if script == nil {
		return fmt.Errorf(Lang(`erropen`, name))
	}
	if script.embedded {
		return fmt.Errorf(Lang(`errembed`))
	}
	if err := checkDep(name, script.Settings.Title); err != nil {
		return err
	}
	delScript(name)
	return SaveStorage()
}
