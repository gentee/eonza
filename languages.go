// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

var (
	langs = map[string]string{
		`en`: `English`,
		`ru`: `Русский`,
	}
	curLang string
	langRes = make(map[string]string)
	mainRes = make(map[string]string)
)

// InitLang loads language resources
func InitLang(lng string) {
	curLang = lng
	if err := yaml.Unmarshal(LanguageAsset(appInfo.Lang), &mainRes); err != nil {
		golog.Fatal(err)
	}
	if lng == appInfo.Lang {
		langRes = mainRes
		return
	}
	if err := yaml.Unmarshal(LanguageAsset(lng), &langRes); err != nil {
		golog.Fatal(err)
	}
	for key, val := range mainRes {
		if _, exist := langRes[key]; !exist {
			langRes[key] = val
		}
	}
}

func Lang(res string, params ...interface{}) string {
	var (
		ok  bool
		val string
	)
	if val, ok = langRes[res]; !ok {
		return res
	}
	if len(params) > 0 {
		return fmt.Sprintf(val, params...)
	}
	return val
}
