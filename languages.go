// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

const (
	LangChar = '%'
)

var (
	langs   = []string{`en`, `ru`}
	langRes []map[string]string
)

// InitLang loads language resources
func InitLang() {
	langRes = make([]map[string]string, len(langs))
	for i, lang := range langs {
		res := make(map[string]string, 32)
		if err := yaml.Unmarshal(LanguageAsset(lang), &res); err != nil {
			golog.Fatal(err)
		}
		langRes[i] = res
	}
}

func GetIdLang(user *User) (ret int) {
	if user == nil && IsScript {
		return scriptTask.Header.Lang
	}
	if u, ok := userSettings[user.ID]; ok {
		for i, lang := range langs {
			if lang == u.Lang {
				ret = i
				break
			}
		}
	}
	return
}

func Lang(idlang int, res string, params ...interface{}) string {
	var (
		ok  bool
		val string
	)
	if idlang > len(langs) {
		idlang = 0
	}
	if val, ok = langRes[idlang][res]; !ok {
		if idlang > 0 {
			if val, ok = langRes[0][res]; !ok {
				return res
			}
		} else {
			return res
		}
	}
	if len(params) > 0 {
		return fmt.Sprintf(val, params...)
	}
	return val
}

func RenderLang(input []rune, idLang int) string {
	var (
		isName bool
	)
	result := make([]rune, 0, len(input))
	name := make([]rune, 0, 32)

	clearName := func() {
		isName = false
		name = name[:0]
	}
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch == LangChar {
			if isName {
				var val string
				val = Lang(idLang, string(name[1:]))
				if len(val) > 0 {
					result = append(result, []rune(val)...)
				} else {
					result = append(result, append(name, LangChar)...)
				}
				clearName()
			} else {
				isName = true
				name = append(name, LangChar)
			}
		} else if isName {
			switch ch {
			case '.':
				name = append(name, ch)
			case ' ', '\n', ',':
				result = append(result, name...)
				clearName()
				result = append(result, ch)
			default:
				if len(name) >= 32 {
					result = append(result, name...)
					result = append(result, ch)
				} else {
					name = append(name, ch)
				}
			}
		} else {
			result = append(result, ch)
		}
	}
	if isName {
		result = append(result, name...)
	}
	return string(result)
}
