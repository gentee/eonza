// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path"
	"regexp"

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
	if ind, ok := mapScripts[name]; ok {
		if ind < len(sysScripts) {
			return &sysScripts[ind]
		}
		return &storage.Scripts[ind-len(sysScripts)]
	}
	return nil
}

// AddHistory save the script in the history
func AddHistory(name string) {
	var i int
	for ; i < HistoryLimit; i++ {
		if storage.History[i] == name {
			storage.History[i] = storage.History[storage.HistoryOff]
			break
		}
	}
	storage.History[storage.HistoryOff] = name
	storage.HistoryOff++
	if storage.HistoryOff == HistoryLimit {
		storage.HistoryOff = 0
	}
}

// GetHistory returns the history list
func GetHistory() []string {
	ret := make([]string, 0)
	for i := storage.HistoryOff - 1; i >= 0; i-- {
		if len(storage.History[i]) > 0 {
			ret = append(ret, storage.History[i])
		}
	}
	for i := HistoryLimit - 1; i >= storage.HistoryOff; i-- {
		if len(storage.History[i]) > 0 {
			ret = append(ret, storage.History[i])
		}
	}
	return ret
}

// LatestScript returns the latest open project
func LatestScript() (ret string) {
	history := GetHistory()
	if len(history) > 0 {
		ret = history[0]
	}
	return
}

func (script *Script) Validate() error {
	re, _ := regexp.Compile(`^[a-z\d\._-]+$`)
	if !re.MatchString(script.Settings.Name) {
		return fmt.Errorf(Lang(`invalidfield`), Lang(`uniquename`))
	}
	if len(script.Settings.Title) == 0 {
		return fmt.Errorf(Lang(`invalidfield`), Lang(`title`))
	}
	return nil
}
