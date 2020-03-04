// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

var mapScripts = make(map[string]int)

type scriptSettings struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type Script struct {
	Settings scriptSettings `json:"settings"`
}

func LoadScripts() {
	for i, item := range storage.Scripts {
		mapScripts[item.Settings.Name] = i
	}
}

func GetScript(name string) *Script {
	if ind, ok := mapScripts[name]; ok {
		return &storage.Scripts[ind]
	}
	return nil
}
