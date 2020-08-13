// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gentee/gentee"
	"gopkg.in/yaml.v2"
)

const (
	LOG_DISABLE = iota
	LOG_ERROR
	LOG_WARN
	LOG_FORM
	LOG_INFO
	LOG_DEBUG
	LOG_INHERIT

	VarChar   = '#'
	VarLength = 32
	VarDeep   = 16

	ErrVarLoop  = `%s variable refers to itself`
	ErrVarDeep  = `maximum depth reached`
	ErrVarConst = `the '%s' constant cannot be modified`
)

type ConditionItem struct {
	Var   string `json:"var,omitempty"`
	Not   bool   `json:"not,omitempty"`
	Cmp   string `json:"cmp,omitempty"`
	Value string `json:"value,omitempty"`
	Next  string `json:"next,omitempty"`

	result bool
}

type FormInfo struct {
	ChResponse chan bool
	Data       string
	ID         uint32
}

type Data struct {
	LogLevel int64
	Vars     []map[string]string
	Mutex    sync.Mutex
	chLogout chan string
	chForm   chan FormInfo
	Global   *map[string]string
}

var (
	formID     uint32
	dataScript Data
	customLib  = []gentee.EmbedItem{
		{Prototype: `init()`, Object: Init},
		{Prototype: `initcmd(str)`, Object: InitCmd},
		{Prototype: `deinit()`, Object: Deinit},
		{Prototype: `Condition(str,str) bool`, Object: Condition},
		{Prototype: `Form(str)`, Object: Form},
		{Prototype: `IsVar(str) bool`, Object: IsVar},
		{Prototype: `LogOutput(int,str)`, Object: LogOutput},
		{Prototype: `Macro(str) str`, Object: Macro},
		{Prototype: `SetLogLevel(int) int`, Object: SetLogLevel},
		{Prototype: `SetYamlVars(str)`, Object: SetYamlVars},
		{Prototype: `SetVar(str,str)`, Object: SetVar},
		{Prototype: `SetVar(str,int)`, Object: SetVarInt},
		{Prototype: `GetVar(str) str`, Object: GetVar},
		{Prototype: `GetVarBool(str) bool`, Object: GetVarBool},
		// For gentee
		{Prototype: `YamlToMap(str) map`, Object: YamlToMap},
	}
)

func IsCond(item *ConditionItem) (err error) {
	var (
		i      int64
		val, s string
	)
	if len(item.Var) == 0 {
		return fmt.Errorf(`empty variable in If Statement`)
	}
	if val, err = Macro(item.Value); err != nil {
		return
	}
	switch item.Cmp {
	case `equal`:
		if len(item.Value) == 0 {
			if i, err = GetVarBool(item.Var); err != nil {
				return
			}
			item.result = i == 0
		} else {
			if s, err = GetVar(item.Var); err != nil {
				return
			}
			item.result = s == val
		}
	default:
		return fmt.Errorf(`Unknown comparison type: %s`, item.Cmp)
	}
	if item.Not {
		item.result = !item.result
	}
	return
}

func Condition(casevar, list string) (ret int64, err error) {
	if len(casevar) > 0 {
		var used int64
		if used, err = GetVarBool(casevar); err != nil || used != 0 {
			return
		}
	}
	var cond []ConditionItem
	if err = json.Unmarshal([]byte(list), &cond); err != nil {
		return
	}
	count := len(cond)
	if count == 0 {
		ret = 1
	} else {
		if err = IsCond(&cond[0]); err != nil {
			return
		}
		// collect OR
		var (
			modeOr bool // = cond[0].Next == `1`
		)
		for i := 1; i < count; i++ {
			modeOr = cond[i-1].Next == `1`
			if !cond[0].result && !modeOr {
				break
			}
			if modeOr {
				if cond[0].result {
					continue
				}
			}
			if err = IsCond(&cond[i]); err != nil {
				return
			}
			cond[0].result = cond[i].result
		}
		if cond[0].result {
			ret = 1
		}
	}

	if ret != 0 && len(casevar) > 0 {
		if err = SetVarInt(casevar, 1); err != nil {
			return
		}
	}
	return
}

func Deinit() {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	dataScript.Vars = dataScript.Vars[:len(dataScript.Vars)-1]
}

func GetVar(name string) (ret string, err error) {
	if IsVar(name) {
		id := len(dataScript.Vars) - 1
		ret, err = Macro(dataScript.Vars[id][name])
	}
	return
}

func GetVarBool(name string) (ret int64, err error) {
	var tmp string
	if tmp, err = GetVar(name); err == nil {
		if len(tmp) > 0 && tmp != `0` && tmp != `false` {
			ret = 1
		}
	}
	return
}

func Init(pars ...interface{}) {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	ind := len(dataScript.Vars)
	dataScript.Vars = append(dataScript.Vars, make(map[string]string))
	for i := 0; i < len(pars); i += 2 {
		dataScript.Vars[ind][pars[i].(string)] = fmt.Sprint(pars[i+1])
	}
}

func InitCmd(name string, pars ...interface{}) bool {
	params := make([]string, len(pars))
	for i, par := range pars {
		switch par.(type) {
		case string:
			params[i] = `"` + fmt.Sprint(par) + `"`
		default:
			params[i] = fmt.Sprint(par)
		}
	}
	LogOutput(LOG_DEBUG, fmt.Sprintf("=> %s(%s)", name, strings.Join(params, `, `)))
	return true
}

func IsVar(key string) bool {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	_, ret := dataScript.Vars[len(dataScript.Vars)-1][key]
	return ret
}

func Form(data string) {
	ch := make(chan bool)
	var dataList []map[string]interface{}

	if json.Unmarshal([]byte(data), &dataList) == nil {
		for i, item := range dataList {
			val, _ := Macro(dataScript.Vars[len(dataScript.Vars)-1][fmt.Sprint(item["var"])])
			dataList[i]["value"] = val
			val, _ = Macro(fmt.Sprint(item["text"]))
			dataList[i]["text"] = val
		}
		if out, err := json.Marshal(dataList); err == nil {
			data = string(out)
		}
	}
	dataScript.Mutex.Lock()
	form := FormInfo{
		ChResponse: ch,
		Data:       data,
		ID:         formID,
	}
	formID++
	dataScript.Mutex.Unlock()
	dataScript.chForm <- form
	<-ch
}

func LogOutput(level int64, message string) {
	var mode = []string{``, `ERROR`, `WARN`, `FORM`, `INFO`, `DEBUG`}
	if level < LOG_ERROR || level > LOG_DEBUG {
		return
	}
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	if level > dataScript.LogLevel {
		return
	}
	dataScript.chLogout <- fmt.Sprintf("[%s] %s %s",
		mode[level], time.Now().Format(`2006/01/02 15:04:05`), message)
}

func replace(values map[string]string, input []rune, stack *[]string,
	glob *map[string]string) ([]rune, error) {
	if len(input) == 0 || strings.IndexRune(string(input), VarChar) == -1 {
		return input, nil
	}
	var (
		err        error
		isName, ok bool
		value      string
		tmp        []rune
	)
	result := make([]rune, 0, len(input))
	name := make([]rune, 0, VarLength+1)

	for i := 0; i < len(input); i++ {
		r := input[i]
		if r != VarChar {
			if isName {
				name = append(name, r)
				if len(name) > VarLength {
					result = append(append(result, VarChar), name...)
					isName = false
					name = name[:0]
				}
			} else {
				result = append(result, r)
			}
			continue
		}
		if isName {
			key := string(name)
			if key[0] == '.' {
				if glob != nil {
					value, ok = (*glob)[key[1:]]
				}
			} else {
				if values != nil {
					value, ok = values[key]
				}
			}
			if ok {
				if len(*stack) < VarDeep {
					for _, item := range *stack {
						if item == key {
							return result, fmt.Errorf(ErrVarLoop, item)
						}
					}
				} else {
					return result, fmt.Errorf(ErrVarDeep)
				}
				*stack = append(*stack, key)
				if tmp, err = replace(values, []rune(value), stack, glob); err != nil {
					return result, err
				}
				*stack = (*stack)[:len(*stack)-1]
				result = append(result, tmp...)
			} else {
				result = append(append(result, VarChar), name...)
				i--
			}
			name = name[:0]
		}
		isName = !isName
	}
	if isName {
		result = append(append(result, VarChar), name...)
	}
	return result, nil
}

func Macro(in string) (string, error) {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	stack := make([]string, 0)
	out, err := replace(dataScript.Vars[len(dataScript.Vars)-1], []rune(in), &stack, dataScript.Global)
	return string(out), err
}

func SetLogLevel(level int64) int64 {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	ret := dataScript.LogLevel
	if level >= LOG_DISABLE && level < LOG_INHERIT {
		dataScript.LogLevel = level
	}
	return ret
}

func SetVar(name, value string) error {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	id := len(dataScript.Vars) - 1
	if strings.HasPrefix(name, `.`) {
		return fmt.Errorf(ErrVarConst, name)
	}
	dataScript.Vars[id][name] = value
	return nil
}

func SetVarInt(name string, value int64) error {
	return SetVar(name, fmt.Sprint(value))
}

func SetYamlVars(in string) error {
	var (
		err error
		tmp map[string]string
	)
	if err = yaml.Unmarshal([]byte(in), &tmp); err != nil {
		return err
	}
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	for name, value := range tmp {
		dataScript.Vars[len(dataScript.Vars)-1][name] = value
	}
	return nil
}

func InitData(chLogout chan string, chForm chan FormInfo, glob *map[string]string) {
	dataScript.Vars = make([]map[string]string, 0, 8)
	dataScript.chLogout = chLogout
	dataScript.chForm = chForm
	dataScript.Global = glob
}

func InitEngine() error {
	return gentee.Customize(&gentee.Custom{
		Embedded: customLib,
	})
}
