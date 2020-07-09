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
		{Prototype: `Form(str)`, Object: Form},
		{Prototype: `IsVar(str) bool`, Object: IsVar},
		{Prototype: `LogOutput(int,str)`, Object: LogOutput},
		{Prototype: `Macro(str) str`, Object: Macro},
		{Prototype: `SetLogLevel(int) int`, Object: SetLogLevel},
		{Prototype: `SetYamlVars(str)`, Object: SetYamlVars},
		{Prototype: `SetVar(str,str)`, Object: SetVar},
		{Prototype: `GetVar(str) str`, Object: GetVar},
		// For gentee
		{Prototype: `YamlToMap(str) map`, Object: YamlToMap},
	}
)

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

func Init() {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	dataScript.Vars = append(dataScript.Vars, make(map[string]string))
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
