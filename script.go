// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"fmt"
	"path"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

var (
	scripts map[string]*Script
)

type scriptSettings struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type Script struct {
	Settings scriptSettings `json:"settings"`

	embedded bool // Embedded script
}

func InitScripts() {
	scripts = make(map[string]*Script)
	for _, tpl := range _escDirs["../eonza-assets/scripts"] {
		var script Script
		fname := tpl.Name()
		data := FileAsset(path.Join(`scripts`, fname))
		if err := yaml.Unmarshal(data, &script); err != nil {
			golog.Fatal(err)
		}
		script.embedded = true
		scripts[script.Settings.Name] = &script
	}
	for name, item := range storage.Scripts {
		scripts[name] = item
	}
}

func (script *Script) Validate() error {
	if !lib.ValidateSysName(script.Settings.Name) {
		return fmt.Errorf(Lang(`invalidfield`), Lang(`uniquename`))
	}
	if len(script.Settings.Title) == 0 {
		return fmt.Errorf(Lang(`invalidfield`), Lang(`title`))
	}
	return nil
}

func ScriptDependences(name string) []ScriptItem {
	var ret []ScriptItem

	// TODO: enumerate all commands
	return ret
}

func (script *Script) SaveScript(original string) error {
	if script.embedded {
		// TODO: error
	}
	if len(original) > 0 && original != script.Settings.Name {
		if _, ok := scripts[script.Settings.Name]; ok {
			// TODO: error exists
		}
		if deps := ScriptDependences(original); len(deps) > 0 {
			// TODO: error dependences
		}
		delete(scripts, original)
		delete(storage.Scripts, original)
	}
	scripts[script.Settings.Name] = script
	storage.Scripts[script.Settings.Name] = script
	return SaveStorage()
}

func DeleteScript(name string) error {
	script := scripts[name]
	if script == nil {
		return fmt.Errorf(Lang(`erropen`, name))
	}
	if script.embedded {
		// TODO: error
	}
	if deps := ScriptDependences(name); len(deps) > 0 {
		// TODO: error dependences
	}
	delete(scripts, name)
	delete(storage.Scripts, name)
	return SaveStorage()
}
