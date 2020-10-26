// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"eonza/lib"
	"fmt"
	"hash/crc64"
	"reflect"
	"strings"

	es "eonza/script"

	"gopkg.in/yaml.v2"
)

type Source struct {
	Linked      map[string]bool
	Strings     []string
	CRCTable    *crc64.Table
	HashStrings map[uint64]int
	Header      *es.Header
	Counter     int
	Funcs       string
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

func (src *Source) FindStrConst(value string) string {
	var (
		id int
		ok bool
	)
	crc := crc64.Checksum([]byte(value), src.CRCTable)
	if id, ok = src.HashStrings[crc]; !ok {
		id = len(src.Strings)
		src.HashStrings[crc] = id
		src.Strings = append(src.Strings, value)
	}
	return fmt.Sprintf(`STR%d`, id)
}

func (src *Source) Value(value string) string {
	var f string
	if len(value) > 2 && value[0] == '<' && value[len(value)-1] == '>' {
		f = `File`
	} else if strings.ContainsRune(value, es.VarChar) {
		f = `Macro`
	}
	value = src.FindStrConst(value)
	if len(f) > 0 {
		value = fmt.Sprintf("%s(%s)", f, value)
	}
	return value
}

func (src *Source) getTypeValue(script *Script, par es.ScriptParam, value string) (string, string) {
	ptype := `str`
	switch par.Type {
	case es.PCheckbox:
		ptype = `bool`
		if value == `0` || len(value) == 0 {
			value = `false`
		} else if value == `1` {
			value = `true`
		}
		if value != `false` && value != `true` {
			value = src.Value(value) + `?`
		}
	case es.PTextarea, es.PSingleText:
		if script.Settings.Name != SourceCode {
			value = src.Value(value)
		}
	case es.PSelect:
		if len(par.Options.Type) > 0 {
			ptype = par.Options.Type
		} else {
			ptype = `str`
		}
		if ptype == `str` {
			value = src.FindStrConst(value)
		}
	case es.PNumber:
		ptype = `int`
	}
	return ptype, value
}

func (src *Source) ScriptValues(script *Script, node scriptTree) ([]Param, []Param, error) {
	values := make([]Param, 0, len(script.Params))
	var optvalues []Param

	errField := func(field string) error {
		glob := langRes[langsId[src.Header.Lang]]
		return fmt.Errorf(langRes[langsId[src.Header.Lang]][`errfield`],
			es.ReplaceVars(field, script.Langs[src.Header.Lang], &glob),
			es.ReplaceVars(script.Settings.Title, script.Langs[src.Header.Lang], &glob))
	}
	var (
		opt    map[string]interface{}
		params map[string]interface{}
	)
	if optional, ok := node.Values[`_optional`]; ok {
		if v, ok := optional.(string); ok {
			if err := yaml.Unmarshal([]byte(v), &opt); err != nil {
				return nil, nil, err
			}
		}
	}
	if adv, ok := node.Values[`_advanced`]; ok {
		var advanced map[string]interface{}
		if v, ok := adv.(string); ok {
			if err := yaml.Unmarshal([]byte(v), &advanced); err != nil {
				return nil, nil, err
			}
			retypeValues(advanced)
		}
		if v, ok := advanced[`params`]; ok {
			if pars, ok := v.(map[string]interface{}); ok {
				params = pars
			}
		}
	}
	for _, par := range script.Params {
		var (
			ptype, value string
			val          interface{}
		)
		if par.Options.Optional {
			val = opt[par.Name]
			if val == nil {
				continue
			}
			switch v := val.(type) {
			case int, int64:
				if par.Type != es.PNumber {
					val = fmt.Sprintf(`"%d"`, v)
				}
			case bool:
				if par.Type != es.PCheckbox {
					if v {
						val = `"true"`
					} else {
						val = `"false"`
					}
				}
			case string:
				if par.Type == es.PNumber || par.Type == es.PCheckbox {
					val = src.Value(v)
					if par.Type == es.PNumber {
						val = fmt.Sprintf(`int(%s)`, val)
					} else if par.Type == es.PCheckbox {
						val = fmt.Sprintf(`bool(%s)`, val)
					}
				}
			default:
				val = fmt.Sprintf(value)
			}
		} else if v, ok := params[par.Name]; ok {
			val = v
		} else {
			val = node.Values[par.Name]
		}
		if val != nil {
			value = strings.TrimSpace(fmt.Sprint(val))
		} else {
			value = par.Options.Default
		}
		ptype, value = src.getTypeValue(script, par, value)
		switch par.Type {
		case es.PTextarea, es.PSingleText, es.PNumber:
			if len(value) == 0 {
				if par.Options.Required {
					return nil, nil, errField(par.Title)
				}
			}
		case es.PList:
			if val != nil && reflect.TypeOf(val).Kind() == reflect.Slice &&
				reflect.ValueOf(val).Len() > 0 {
				out, err := json.Marshal(val)
				if err != nil {
					return nil, nil, err
				}
				value = src.FindStrConst(string(out))
			} else {
				if par.Options.Required {
					return nil, nil, errField(par.Title)
				}
				value = src.FindStrConst(`[]`)
			}
		}
		param := Param{
			Value: value,
			Type:  ptype,
			Name:  par.Name,
		}
		if par.Options.Optional {
			optvalues = append(optvalues, param)
		} else {
			values = append(values, param)
		}
	}
	return values, optvalues, nil
}

func (src *Source) Predefined(script *Script) (ret string, err error) {
	if len(script.Langs[LangDefCode]) > 0 {
		var data []byte
		predef := make(map[string]string)

		for name, value := range script.Langs[LangDefCode] {
			if !strings.HasPrefix(name, `_`) {
				predef[name] = value
			}
		}
		if src.Header.Lang != LangDefCode {
			for name, value := range script.Langs[src.Header.Lang] {
				if !strings.HasPrefix(name, `_`) {
					predef[name] = value
				}
			}
		}
		data, err = yaml.Marshal(predef)
		if err != nil {
			return
		}
		ret = `SetYamlVars(` + src.FindStrConst(string(data)) + ")\r\n"
	}
	return
}

func processIf(input string) string {
	var (
		out    []rune
		isName bool
		off    int
	)
	in := []rune(input)
	for i := 0; i < len(in); i++ {
		ch := in[i]
		if isName {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
				ch == '_' || ch == '.' {
				continue
			}
			name := fmt.Sprintf(`GetVarBool("%s")`, string(in[off:i]))
			out = append(out, []rune(name)...)
			isName = false
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			off = i
			isName = true
			continue
		}
		switch ch {
		case ' ', '!', '&', '|', '\t':
			out = append(out, ch)
		default:
			return input
		}
	}
	if isName {
		name := fmt.Sprintf(`GetVarBool("%s")`, string(in[off:]))
		out = append(out, []rune(name)...)
	}
	return string(out)
}

func (src *Source) Script(node scriptTree) (string, error) {
	var (
		ifcond string
	)
	script := getScript(node.Name)
	if script == nil {
		return ``, fmt.Errorf(Lang(DefLang, `erropen`), node.Name)
	}
	idname := lib.IdName(script.Settings.Name)
	values, optvalues, err := src.ScriptValues(script, node)
	if err != nil {
		return ``, err
	}
	if ifraw, ok := node.Values[`_ifcond`]; ok {
		ifcond, _ = ifraw.(string)
		ifcond = processIf(ifcond)
	}
	var params []string
	if !src.Linked[idname] || script.Settings.Name == SourceCode || len(node.Children) > 0 {
		src.Linked[idname] = true

		tmp, err := src.Tree(node.Children)
		if err != nil {
			return ``, err
		}
		var (
			code, predef string
		)
		if predef, err = src.Predefined(script); err != nil {
			return ``, err
		}
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
			}
		}
		if script.Settings.Name == SourceCode || len(node.Children) > 0 {
			idname = fmt.Sprintf("%s%d", idname, src.Counter)
			src.Counter++
		}
		code = strings.TrimRight(code, "\r\n")
		var parNames, prefix, suffix, initcmd string
		if script.Settings.Name != SourceCode {
			var vars []string
			for _, par := range values {
				params = append(params, fmt.Sprintf("%s %s", par.Type, par.Name))
				parNames += `,` + par.Name
				vars = append(vars, fmt.Sprintf(`"%s", %[1]s`, par.Name))
			}
			/*			// Now log info is without optional parameters
						for _, par := range optvalues {
							parNames += `,` + par.Name
						}*/
			for _, par := range script.Params {
				if !par.Options.Optional {
					continue
				}
				ptype, def := src.getTypeValue(script, par, par.Options.Default)
				if len(def) > 0 {
					def = ` = ` + def
				}
				vars = append(vars, fmt.Sprintf(`"%s", %[1]s`, par.Name))
				prefix += fmt.Sprintf("%s ?%s%s\r\n", ptype, par.Name, def)
			}
			if len(script.Tree) > 0 {
				code += "\r\ninit(" + strings.Join(vars, `,`) + ")\r\n" + predef
				tmp, err = src.Tree(script.Tree)
				if err != nil {
					return ``, err
				}
				code += "\r\n" + tmp
				code += "\r\ndeinit()"
			}
		}
		if script.Settings.LogLevel < es.LOG_INHERIT {
			prefix += fmt.Sprintf("int prevLog = SetLogLevel(%d)\r\n", script.Settings.LogLevel)
			suffix = "\r\nSetLogLevel(prevLog)"
		}
		name := script.Settings.Name
		if script.Settings.LogLevel == es.LOG_INFO {
			name = `*` + name
		}
		initcmd = fmt.Sprintf("initcmd(`%s`%s)\r\n", name, parNames)
		/*		if len(script.Tree) > 0 || len(predef) > 0 {
				initcmd += "init()\r\n" + predef
				code += "\r\ndeinit()"
			}*/
		code = initcmd + code
		src.Funcs += fmt.Sprintf("func %s(%s) {\r\n", idname, strings.Join(params, `,`)) +
			prefix + code + suffix + "\r\n}\r\n"
	}
	params = params[:0]
	if script.Settings.Name != SourceCode {
		for _, par := range values {
			params = append(params, par.Value)
		}
		for _, par := range optvalues {
			params = append(params, fmt.Sprintf("%s: %s", par.Name, par.Value))
		}
	}
	out := fmt.Sprintf("   %s(%s)\r\n", idname, strings.Join(params, `,`))
	if len(ifcond) > 0 {
		out = fmt.Sprintf(`   if %s {
        %s   }`, ifcond, out)
	}
	return out, nil
}

func ValToStr(input string) string {
	var out string

	if strings.ContainsAny(input, "`%$") {
		out = strings.ReplaceAll(input, `\`, `\\`)
		out = `"` + strings.ReplaceAll(out, `"`, `\"`) + `"`
	} else {
		out = "`" + input + "`"
	}
	return out
}

func GenSource(script *Script, header *es.Header) (string, error) {
	var params string
	src := &Source{
		Linked:      make(map[string]bool),
		CRCTable:    crc64.MakeTable(crc64.ISO),
		HashStrings: make(map[uint64]int),
		Header:      header,
	}
	values, optvalues, err := src.ScriptValues(script, scriptTree{})
	if err != nil {
		return ``, err
	}
	for _, par := range append(values, optvalues...) {
		val := par.Value
		if par.Type == `str` {
			val = ValToStr(val)
		}
		params += fmt.Sprintf("%s %s = %s\r\n", par.Type, par.Name, val)
	}
	level := storage.Settings.LogLevel
	if script.Settings.LogLevel < es.LOG_INHERIT {
		level = script.Settings.LogLevel
	}
	params += fmt.Sprintf("SetLogLevel(%d)\r\ninit()\r\n", level)
	code := strings.TrimSpace(strings.ReplaceAll(script.Code, `%body%`, ``))
	if len(code) > 0 {
		code += "\r\n"
	}
	if predef, err := src.Predefined(script); err != nil {
		return ``, err
	} else {
		code = predef + code
	}
	body, err := src.Tree(script.Tree)
	if err != nil {
		return ``, err
	}
	var constStr string
	if len(src.Strings) > 0 {
		constStr = "const {\r\n"
		for i, val := range src.Strings {
			constStr += fmt.Sprintf("STR%d = %s\r\n", i, ValToStr(val))
		}
		constStr += "}\r\n"
	}
	constStr += `const IOTA { LOG_DISABLE
	LOG_ERROR LOG_WARN LOG_FORM LOG_INFO LOG_DEBUG }
`
	return fmt.Sprintf("%s%s\r\nrun {\r\n%s%s%s\r\ndeinit()}", constStr, src.Funcs, params,
		code, body), nil
}
