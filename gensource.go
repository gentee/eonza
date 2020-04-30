// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"fmt"
	"strings"
)

type Source struct {
	Linked  map[string]bool
	Counter int
	Funcs   string
}

type Param struct {
	Value string
	Type  string
	Name  string
}

func (src *Source) Tree(tree []scriptTree) (string, error) {
	var (
		body, tmp string
		err       error
	)
	for _, child := range tree {
		if child.Disable {
			continue
		}
		if tmp, err = src.Script(child); err != nil {
			return ``, err
		}
		body += tmp
	}
	return body, nil
}

func ScriptValues(script *Script, node scriptTree) ([]Param, error) {
	values := make([]Param, 0, len(script.Params))
	for _, par := range script.Params {
		var (
			ptype, value string
		)
		val := node.Values[par.Name]
		if val != nil {
			value = strings.TrimSpace(fmt.Sprint(val))
		}
		switch par.Type {
		case PCheckbox:
			ptype = `bool`
			if value == `false` || value == `0` || len(value) == 0 {
				value = `false`
			} else {
				value = `true`
			}
		case PTextarea:
			ptype = `str`
			if len(value) == 0 {
				if par.options.Required {
					return nil, fmt.Errorf("The '%s' field is required in the '%s' command", par.Title,
						script.Settings.Title)
				}
				value = par.options.Default
			}
		}
		values = append(values, Param{
			Value: value,
			Type:  ptype,
			Name:  par.Name,
		})
	}
	return values, nil
}

func (src *Source) Script(node scriptTree) (string, error) {
	script := getScript(node.Name)
	if script == nil {
		return ``, fmt.Errorf(Lang(`erropen`), node.Name)
	}
	idname := lib.IdName(script.Settings.Name)
	values, err := ScriptValues(script, node)
	if err != nil {
		return ``, err
	}
	if !src.Linked[idname] || script.Settings.Name == SourceCode {
		src.Linked[idname] = true
		tmp, err := src.Tree(node.Children)
		if err != nil {
			return ``, err
		}
		var (
			code string
		)
		if script.Settings.Name == SourceCode {
			code = values[1].Value
		} else {
			code = script.Code
		}
		code = strings.ReplaceAll(code, `%body%`, tmp)
		if script.Settings.Name == SourceCode {
			if values[0].Value == `true` {
				src.Funcs += code + "\r\n"
				return ``, nil
			} else {
				idname = fmt.Sprintf("%s%d", idname, src.Counter)
				src.Counter++
			}
		}
		src.Funcs += fmt.Sprintf("func %s {\r\n", idname) +
			strings.TrimRight(code, "\r\n") + "\r\n}\r\n"
	}
	return fmt.Sprintf("   %s()\r\n", idname), nil
}

func GenSource(script *Script) (string, error) {
	src := &Source{
		Linked: make(map[string]bool),
	}
	body, err := src.Tree(script.Tree)
	if err != nil {
		return ``, err
	}
	return fmt.Sprintf("%s\r\nrun {\r\n%s}", src.Funcs, body), nil
}
