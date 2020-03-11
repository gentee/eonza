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
	sysScripts []Script
	mapScripts map[string]int
)

type scriptSettings struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type Script struct {
	Settings scriptSettings `json:"settings"`
}

func InitScripts() {
	sysScripts = make([]Script, 0, 64)
	mapScripts = make(map[string]int)
	for _, tpl := range _escDirs["../eonza-assets/scripts"] {
		var script Script
		fname := tpl.Name()
		data := FileAsset(path.Join(`scripts`, fname))
		if err := yaml.Unmarshal(data, &script); err != nil {
			golog.Fatal(err)
		}
		mapScripts[script.Settings.Name] = len(sysScripts)
		sysScripts = append(sysScripts, script)
	}
	off := len(sysScripts)
	for i, item := range storage.Scripts {
		mapScripts[item.Settings.Name] = off + i
	}
}

func GetScript(name string) *Script {
	mutex.RLock()
	defer mutex.RUnlock()
	if ind, ok := mapScripts[name]; ok {
		if ind < len(sysScripts) {
			return &sysScripts[ind]
		}
		return &storage.Scripts[ind-len(sysScripts)]
	}
	return nil
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
