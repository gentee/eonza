// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"encoding/json"
	"eonza/lib"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gentee/gentee"
	"github.com/gentee/gentee/vm"
	"gopkg.in/yaml.v2"
)

type ParamType int

const (
	PCheckbox ParamType = iota
	PTextarea
	PSingleText
	PSelect
	PNumber
	PList
	PHTMLText
	PButton
	PDynamic
)

type ScriptItem struct {
	Title string `json:"title" yaml:"title"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

type ScriptParam struct {
	Name    string        `json:"name" yaml:"name"`
	Title   string        `json:"title" yaml:"title"`
	Type    ParamType     `json:"type" yaml:"type"`
	Options ScriptOptions `json:"options,omitempty" yaml:"options,omitempty"`
}

type FormParam struct {
	Var     string `json:"var,omitempty"`
	Text    string `json:"text,omitempty"`
	Value   string `json:"value,omitempty"`
	Type    string `json:"type"`
	Options string `json:"options,omitempty"`
}

type ScriptOptions struct {
	Initial  string        `json:"initial,omitempty" yaml:"initial,omitempty"`
	Default  string        `json:"default,omitempty" yaml:"default,omitempty"`
	Required bool          `json:"required,omitempty" yaml:"required,omitempty"`
	Optional bool          `json:"optional,omitempty" yaml:"optional,omitempty"`
	Type     string        `json:"type,omitempty" yaml:"type,omitempty"`
	Items    []ScriptItem  `json:"items,omitempty" yaml:"items,omitempty"`
	List     []ScriptParam `json:"list,omitempty" yaml:"list,omitempty"`
	Output   []string      `json:"output,omitempty" yaml:"output,omitempty"`
	// for Form command
	If string `json:"if,omitempty" yaml:"if,omitempty"`
}

const (
	LOG_DISABLE = iota
	LOG_ERROR
	LOG_WARN
	LOG_FORM
	LOG_INFO
	LOG_DEBUG
	LOG_INHERIT

	VarChar   = '#'
	VarLength = 48
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
	ObjVars  []sync.Map
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
		{Prototype: `File(str) str`, Object: FileLoad},
		{Prototype: `Form(str)`, Object: Form},
		{Prototype: `IsObjVar(str) bool`, Object: IsVarObj},
		{Prototype: `IsVar(str) bool`, Object: IsVar},
		{Prototype: `LogOutput(int,str)`, Object: LogOutput},
		{Prototype: `Macro(str) str`, Object: Macro},
		{Prototype: `ResultVar(str,str)`, Object: ResultVar},
		{Prototype: `ResultVar(str,obj)`, Object: ResultVarObj},
		{Prototype: `SetLogLevel(int) int`, Object: SetLogLevel},
		{Prototype: `SetYamlVars(str)`, Object: SetYamlVars},
		{Prototype: `SetVar(str,bool)`, Object: SetVarBool},
		{Prototype: `SetVar(str,str)`, Object: SetVar},
		{Prototype: `SetVar(str,int)`, Object: SetVarInt},
		{Prototype: `SetVar(str,obj)`, Object: SetVarObj},
		{Prototype: `GetVar(str) str`, Object: GetVar},
		{Prototype: `GetVarBool(str) bool`, Object: GetVarBool},
		{Prototype: `GetVarInt(str) int`, Object: GetVarInt},
		{Prototype: `GetVarObj(str) obj`, Object: GetVarObj},
		// For gentee
		{Prototype: `YamlToMap(str) map`, Object: YamlToMap},
	}
)

func IsCond(rt *vm.Runtime, item *ConditionItem) (err error) {
	var (
		i      int64
		val, s string
	)
	if len(item.Var) == 0 && len(item.Value) == 0 {
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
	case `fileexists`:
		if len(item.Var) > 0 {
			if s, err = GetVar(item.Var); err != nil {
				return
			}
		} else {
			s = val
		}
		if i, err = vm.ExistFile(rt, s); err != nil {
			return
		}
		item.result = i != 0
	case `envexists`:
		if len(item.Var) > 0 {
			if s, err = GetVar(item.Var); err != nil {
				return
			}
		} else {
			s = val
		}
		_, item.result = os.LookupEnv(s)
	case `match`:
		if s, err = GetVar(item.Var); err != nil {
			return
		}
		i, err = vm.MatchºStrStr(s, val)
		item.result = i != 0
	default:
		return fmt.Errorf(`Unknown comparison type: %s`, item.Cmp)
	}
	if item.Not {
		item.result = !item.result
	}
	return
}

func Condition(rt *vm.Runtime, casevar, list string) (ret int64, err error) {
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
		if err = IsCond(rt, &cond[0]); err != nil {
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
			if err = IsCond(rt, &cond[i]); err != nil {
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
	dataScript.ObjVars = dataScript.ObjVars[:len(dataScript.ObjVars)-1]
}

func FileLoad(rt *vm.Runtime, fname string) (ret string, err error) {
	if ret, err = Macro(fname); err != nil {
		return
	}
	isValue := len(ret) > 2 && ret[0] == '<' && ret[len(ret)-1] == '>'
	if isValue {
		if len(ret) > 256 || strings.Contains(ret, "\n") {
			return
		}
		ret = ret[1 : len(ret)-1]
	}
	return vm.ReadFileºStr(rt, ret)
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

func GetVarInt(name string) (ret int64, err error) {
	var tmp string
	if tmp, err = GetVar(name); err == nil {
		ret, _ = strconv.ParseInt(tmp, 10, 64)
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
	dataScript.ObjVars = append(dataScript.ObjVars, sync.Map{})
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
	level := int64(LOG_DEBUG)
	info := name[0] == '*'
	if info {
		name = name[1:]
		level = LOG_INFO
	}
	msg := fmt.Sprintf("=> %s(%s)", name, strings.Join(params, `, `))
	LogOutput(level, msg)
	return true
}

func IsVar(key string) bool {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	_, ret := dataScript.Vars[len(dataScript.Vars)-1][key]
	return ret
}

func loadForm(data string, form *[]map[string]interface{}) error {

	var dataList []map[string]interface{}

	ifcond := func(ifval string) (ret bool, err error) {
		var (
			not  bool
			iret int64
		)
		if len(ifval) == 0 {
			return true, nil
		}
		if ifval[0] == '!' {
			not = true
			ifval = ifval[1:]
		}
		if iret, err = GetVarBool(ifval); err != nil {
			return
		}
		ret = iret != 0
		if not {
			ret = !ret
		}
		return
	}
	pbutton := fmt.Sprint(PButton)
	pdynamic := fmt.Sprint(PDynamic)
	if json.Unmarshal([]byte(data), &dataList) == nil {
		for i, item := range dataList {
			if opt, optok := item["options"]; optok {
				var ifval string
				switch v := opt.(type) {
				case string:
					var options ScriptOptions
					if json.Unmarshal([]byte(v), &options) == nil {
						ifval = options.If
					}
				case map[string]interface{}:
					if ifdata, ok := v["if"]; ok {
						ifval = fmt.Sprint(ifdata)
					}
				}
				if ifok, err := ifcond(ifval); err != nil {
					return err
				} else if !ifok {
					continue
				}
			}
			varname := fmt.Sprint(item["var"])
			val, _ := Macro(dataScript.Vars[len(dataScript.Vars)-1][varname])
			if item["type"] == pdynamic {
				loadForm(val, form)
				continue
			}
			dataList[i]["value"] = val
			val, _ = Macro(fmt.Sprint(item["text"]))
			dataList[i]["text"] = val
			if item["type"] == pbutton {
				SetVar(varname, ``)
			}
			*form = append(*form, dataList[i])
		}
	}
	return nil
}

func Form(data string) error {
	ch := make(chan bool)
	formList := make([]map[string]interface{}, 0, 32)

	loadForm(data, &formList)
	if len(formList) > 0 {
		if out, err := json.Marshal(formList); err == nil {
			data = string(out)
		}
	}
	dataScript.Mutex.Lock()
	if (*dataScript.Global)[`isconsole`] == `true` {
		url := fmt.Sprintf(`http://localhost:%s`, (*dataScript.Global)[`port`])
		fmt.Println(fmt.Sprintf((*dataScript.Global)["formopen"], url))
		lib.Open(url)
	}

	form := FormInfo{
		ChResponse: ch,
		Data:       data,
		ID:         formID,
	}
	formID++
	dataScript.Mutex.Unlock()
	dataScript.chForm <- form
	<-ch
	return nil
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
		err               error
		isName, ok, isobj bool
		value             string
		tmp               []rune
		qstack            int
	)
	result := make([]rune, 0, len(input))
	name := make([]rune, 0, VarLength+1)
	clearName := func() {
		name = name[:0]
		isobj = false
		qstack = 0
	}
	for i := 0; i < len(input); i++ {
		r := input[i]
		if r != VarChar || qstack > 0 {
			if isName {
				name = append(name, r)
				switch r {
				case ']':
					qstack--
				case '[':
					qstack++
					fallthrough
				case '.':
					isobj = true
				}
				if len(name) > VarLength {
					result = append(append(result, VarChar), name...)
					isName = false
					clearName()
				}
			} else {
				result = append(result, r)
			}
			continue
		}
		if isName {
			ok = false
			key := string(name)
			if len(key) > 0 {
				if key[0] == '.' {
					if glob != nil {
						value, ok = (*glob)[key[1:]]
					}
				} else {
					if values != nil {
						value, ok = values[key]
					}
					if !ok {
						if isobj {
							value, ok = ReplaceObj(key)
						} else {
							value, ok = ObjToStr(key)
						}
					}
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
			clearName()
		}
		isName = !isName
	}
	if isName {
		result = append(append(result, VarChar), name...)
	}
	return result, nil
}

func macro(in string) (string, error) {
	stack := make([]string, 0)
	out, err := replace(dataScript.Vars[len(dataScript.Vars)-1], []rune(in), &stack, dataScript.Global)
	return string(out), err
}

func Macro(in string) (string, error) {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	return macro(in)
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

func setRawVar(shift int, name, value string) error {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	off := len(dataScript.Vars) - 1 - shift
	if strings.HasPrefix(name, `.`) {
		return fmt.Errorf(ErrVarConst, name)
	}
	if off < 0 {
		return fmt.Errorf(`set shift var %s error`, name)
	}
	dataScript.Vars[off][name] = value
	return nil
}

func ResultVar(name, value string) error {
	return setRawVar(1, name, value)
}

func SetVar(name, value string) error {
	return setRawVar(0, name, value)
}

func SetVarInt(name string, value int64) error {
	return SetVar(name, fmt.Sprint(value))
}

func SetVarBool(name string, value int64) error {
	val := `false`
	if value != 0 {
		val = `true`
	}
	return SetVar(name, val)
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
