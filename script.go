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
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v2"
)

var (
	scripts map[string]*Script
)

type ParamType int

const (
	PCheckbox ParamType = iota
	PTextarea
	PSingleText
	PSelect
	PNumber
	PList
)

type scriptSettings struct {
	Name     string `json:"name" yaml:"name"`
	Title    string `json:"title" yaml:"title"`
	Desc     string `json:"desc,omitempty" yaml:"desc,omitempty"`
	LogLevel int    `json:"loglevel" yaml:"loglevel"`
	Unrun    bool   `json:"unrun,omitempty" yaml:"unrun,omitempty"`
	Help     string `json:"help,omitempty" yaml:"help,omitempty"`
	HelpLang string `json:"helplang,omitempty" yaml:"helplang,omitempty"`
}

type scriptItem struct {
	Title string `json:"title" yaml:"title"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

type scriptOptions struct {
	Initial  string        `json:"initial,omitempty" yaml:"initial,omitempty"`
	Default  string        `json:"default,omitempty" yaml:"default,omitempty"`
	Required bool          `json:"required,omitempty" yaml:"required,omitempty"`
	Type     string        `json:"type,omitempty" yaml:"type,omitempty"`
	Items    []scriptItem  `json:"items,omitempty" yaml:"items,omitempty"`
	List     []scriptParam `json:"list,omitempty" yaml:"list,omitempty"`
	Output   []string      `json:"output,omitempty" yaml:"output,omitempty"`
}

type scriptParam struct {
	Name    string        `json:"name" yaml:"name"`
	Title   string        `json:"title" yaml:"title"`
	Type    ParamType     `json:"type" yaml:"type"`
	Options scriptOptions `json:"options,omitempty" yaml:"options,omitempty"`
}

type scriptTree struct {
	Name     string                 `json:"name" yaml:"name"`
	Open     bool                   `json:"open,omitempty" yaml:"open,omitempty"`
	Disable  bool                   `json:"disable,omitempty" yaml:"disable,omitempty"`
	Values   map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	Children []scriptTree           `json:"children,omitempty" yaml:"children,omitempty"`
}

type Script struct {
	Settings scriptSettings               `json:"settings" yaml:"settings"`
	Params   []scriptParam                `json:"params,omitempty" yaml:"params,omitempty"`
	Tree     []scriptTree                 `json:"tree,omitempty" yaml:"tree,omitempty"`
	Langs    map[string]map[string]string `json:"langs,omitempty" yaml:"langs,omitempty"`
	Code     string                       `json:"code,omitempty" yaml:"code,omitempty"`
	folder   bool                         // can have other commands inside
	embedded bool                         // Embedded script
	initial  string                       // Initial value
}

func getScript(name string) (script *Script) {
	return scripts[lib.IdName(name)]
}

func retypeValues(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, item := range v {
			if val := retypeValues(item); val != nil {
				v[key] = val
			}
		}
	case []interface{}:
		for i, item := range v {
			if val := retypeValues(item); val != nil {
				v[i] = val
			}
		}
	case map[interface{}]interface{}:
		ret := make(map[string]interface{})
		for key, item := range v {
			if val := retypeValues(item); val != nil {
				item = val
			}
			ret[fmt.Sprint(key)] = item
		}
		return ret
	}
	return nil
}

func retypeTree(tree []scriptTree) {
	for _, item := range tree {
		retypeValues(item.Values)
		retypeTree(item.Children)
	}
}

func setScript(script *Script) error {
	var ivalues map[string]interface{} //string

	scripts[lib.IdName(script.Settings.Name)] = script
	if len(script.Params) > 0 {
		ivalues = make(map[string]interface{}) //string)
	}
	for _, par := range script.Params {
		if par.Type == PList {
			ivalues[par.Name] = []interface{}{}
		} else if len(par.Options.Initial) > 0 {
			ivalues[par.Name] = par.Options.Initial
		}
	}
	if len(ivalues) > 0 {
		initial, err := json.Marshal(ivalues)
		if err != nil {
			return err
		}
		script.initial = string(initial)
	}
	retypeTree(script.Tree)
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
		if err := setScript(&script); err != nil {
			golog.Fatal(err)
		}
	}
	for name, item := range storage.Scripts {
		if scripts[lib.IdName(name)] != nil {
			golog.Errorf(`The '%s' script has been loaded as embedded script`, name)
			continue
		}
		//
		item.folder = isfolder(item)
		if err := setScript(item); err != nil {
			golog.Fatal(err)
		}
	}
}

func (script *Script) Validate() error {
	if !lib.ValidateSysName(script.Settings.Name) {
		return fmt.Errorf(Lang(DefLang, `invalidfield`), Lang(DefLang, `name`))
	}
	if len(script.Settings.Title) == 0 {
		return fmt.Errorf(Lang(DefLang, `invalidfield`), Lang(DefLang, `title`))
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

func ScriptDependences(c echo.Context, name string) []ScriptItem {
	var ret []ScriptItem

	for _, item := range scripts {
		if item.embedded {
			continue
		}
		if scanTree(item.Tree, name) {
			ret = append(ret, ScriptToItem(c, item))
		}
	}
	return ret
}

func checkDep(c echo.Context, name, title string) error {
	if deps := ScriptDependences(c, name); len(deps) > 0 {
		ret := make([]string, len(deps))
		for i, item := range deps {
			ret[i] = item.Title
		}
		return fmt.Errorf(Lang(DefLang, `depscript`), title, strings.Join(ret, `,`))
	}
	return nil
}

func (script *Script) SaveScript(c echo.Context, original string) error {
	if curScript := getScript(original); curScript != nil && curScript.embedded {
		return fmt.Errorf(Lang(DefLang, `errembed`))
	}
	if len(original) > 0 && original != script.Settings.Name {
		if getScript(script.Settings.Name) != nil {
			return fmt.Errorf(Lang(DefLang, `errscriptname`), script.Settings.Name)
		}
		if err := checkDep(c, original, script.Settings.Title); err != nil {
			return err
		}
		delScript(original)
	}
	script.folder = script.Settings.Name == SourceCode ||
		strings.Contains(script.Code, `%body%`)
	if err := setScript(script); err != nil {
		return err
	}
	storage.Scripts[lib.IdName(script.Settings.Name)] = script
	return SaveStorage()
}

func DeleteScript(c echo.Context, name string) error {
	script := getScript(name)
	if script == nil {
		return fmt.Errorf(Lang(DefLang, `erropen`, name))
	}
	if script.embedded {
		return fmt.Errorf(Lang(DefLang, `errembed`))
	}
	if err := checkDep(c, name, script.Settings.Title); err != nil {
		return err
	}
	delScript(name)
	return SaveStorage()
}
