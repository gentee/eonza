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

	"github.com/atotto/clipboard"
	"github.com/gentee/gentee"
	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
	"gopkg.in/ini.v1"
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
	PPassword

	EonzaDynamic = `eonza.dynamic.constant`
)

type PostNfy struct {
	TaskID uint32
	Text   string `json:"text"`
	Script string
}

type PostScript struct {
	TaskID uint32 `json:"taskid"`
	Script string `json:"script"`
	Data   string `json:"data"`
	Silent bool   `json:"silent"`
	UserID uint32 `json:"userid"`
	RoleID uint32 `json:"roleid"`
}

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
	Flags    string        `json:"flags,omitempty" yaml:"flags,omitempty"`
	Type     string        `json:"type,omitempty" yaml:"type,omitempty"`
	Items    []ScriptItem  `json:"items,omitempty" yaml:"items,omitempty"`
	List     []ScriptParam `json:"list,omitempty" yaml:"list,omitempty"`
	Output   []string      `json:"output,omitempty" yaml:"output,omitempty"`
	// for Form command
	If string `json:"if,omitempty" yaml:"if,omitempty"`
}

type ThreadOptions struct {
	LogLevel int64
	Refs     []string
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
	Ref        string
	ChResponse chan bool
	Data       string
	ID         uint32
}

type Data struct {
	//	LogLevel int64
	Vars     []map[string]string
	ObjVars  []sync.Map
	Mutex    sync.Mutex
	chLogout chan string
	chForm   chan FormInfo
	chReport chan Report
	Global   *map[string]string
}

var (
	Logs = map[string]int{
		`DISABLE`: LOG_DISABLE,
		`ERROR`:   LOG_ERROR,
		`WARN`:    LOG_WARN,
		`FORM`:    LOG_FORM,
		`INFO`:    LOG_INFO,
		`DEBUG`:   LOG_DEBUG,
		`INHERIT`: LOG_INHERIT,
	}
	MainThread *vm.Runtime
	formID     uint32
	dataScript Data
	customLib  = []gentee.EmbedItem{
		{Prototype: `thread(int)`, Object: Thread},
		{Prototype: `pushref(str)`, Object: PushRef},
		{Prototype: `popref()`, Object: PopRef},
		{Prototype: `ref() str`, Object: GetRef},
		{Prototype: `init()`, Object: Init},
		{Prototype: `initcmd(int,str) int`, Object: InitCmd},
		{Prototype: `deinit()`, Object: Deinit},
		{Prototype: `Condition(map.obj) bool`, Object: MapCondition},
		{Prototype: `Condition(str,str) bool`, Object: Condition},
		{Prototype: `CopyClipboard(str)`, Object: CopyClipboard},
		{Prototype: `File(str) str`, Object: FileLoad},
		{Prototype: `Form(str)`, Object: Form},
		{Prototype: `FillForm(str,str)`, Object: FillForm},
		{Prototype: `GetClipboard() str`, Object: GetClipboard},
		{Prototype: `IsEntry() bool`, Object: IsEntry},
		{Prototype: `IsVarObj(str) bool`, Object: IsVarObj},
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
		{Prototype: `SetJsonVar(str,str)`, Object: SetJsonVar},
		{Prototype: `GetVar(str) str`, Object: GetVar},
		{Prototype: `GetVarBool(str) bool`, Object: GetVarBool},
		{Prototype: `GetVarBytes(str) str`, Object: GetVarBytes},
		{Prototype: `GetVarInt(str) int`, Object: GetVarInt},
		{Prototype: `GetVarObj(str) obj`, Object: GetVarObj},
		{Prototype: `GetVarRaw(str) str`, Object: GetVarRaw},
		{Prototype: `GetConst(str) str`, Object: GetConst},
		{Prototype: `SendNotification(str)`, Object: SendNotification},
		{Prototype: `SendEmail(obj, obj)`, Object: SendEmail},
		{Prototype: `SQLClose(str)`, Object: SQLClose},
		{Prototype: `SQLConnection(map.str, str)`, Object: SQLConnection},
		{Prototype: `SQLExec(str,str,arr.str)`, Object: SQLExec},
		{Prototype: `SQLQuery(str,str,arr.str,str)`, Object: SQLQuery},
		{Prototype: `SQLRow(str,str,arr.str,str)`, Object: SQLRow},
		{Prototype: `SQLValue(str,str,arr.str,str)`, Object: SQLValue},
		{Prototype: `ConvertText(str,str,str) str`, Object: ConvertText},
		{Prototype: `MarkdownToHTML(str) str`, Object: lib.Markdown},
		{Prototype: `RunScript(str,str,bool)`, Object: RunScript},
		{Prototype: `LoadIni(buf) handle`, Object: LoadIni},
		{Prototype: `GetIniValue(handle,str,str,str,str) bool`, Object: GetIniValue},
		{Prototype: `CreateReport(str,str)`, Object: CreateReport},
		{Prototype: `AppendToArray(str,str)`, Object: AppendToArray},
		{Prototype: `AppendToMap(str,str,str)`, Object: AppendToMap},

		// Office functions
		{Prototype: `DocxTemplate(str,str)`, Object: DocxTemplate},
		{Prototype: `OdtTemplate(str,str)`, Object: OdtTemplate},
		{Prototype: `OdsTemplate(str,str)`, Object: OdsTemplate},
		{Prototype: `XlsxTemplate(str,str)`, Object: XlsxTemplate},
		{Prototype: `OpenXLSX(str) handle`, Object: OpenXLSX},
		{Prototype: `XlsxSheetName(handle,int) str`, Object: XLSheetName},
		{Prototype: `XlsxRows(handle,str,str) handle`, Object: XLRows},
		{Prototype: `XlsxNextRow(handle) bool`, Object: XLNextRow},
		{Prototype: `XlsxGetRow(handle) obj`, Object: XLGetRow},
		{Prototype: `XlsxGetCell(handle,str,str) str`, Object: XLGetCell},
		// Windows functions
		{Prototype: `RegistrySubkeys(int,str,int) arr.str`, Object: RegistrySubkeys},
		{Prototype: `CreateRegistryKey(int,str,int) handle`, Object: CreateRegistryKey},
		{Prototype: `CloseRegistryKey(handle)`, Object: CloseRegistryKey},
		{Prototype: `SetRegistryValue(handle,str,int,str)`, Object: SetRegistryValue},
		{Prototype: `RegistryValues(int,str,int) arr.str`, Object: RegistryValues},
		{Prototype: `DeleteRegistryKey(int,str,int)`, Object: DeleteRegistryKey},
		{Prototype: `DeleteRegistryValue(handle,str)`, Object: DeleteRegistryValue},
		{Prototype: `GetRegistryValue(handle,str,str) str`, Object: GetRegistryValue},
		{Prototype: `OpenRegistryKey(int,str,int) handle`, Object: OpenRegistryKey},
		// HTML parsing
		{Prototype: `ParseHTML(str) handle`, Object: ParseHTML},
		{Prototype: `FindHTML(handle,str) handle`, Object: FindHTML},
		{Prototype: `AttribHTML(handle,str) str`, Object: AttribHTML},
		{Prototype: `TextHTML(handle) str`, Object: TextHTML},
		{Prototype: `ChildrenHTML(handle) arr.handle`, Object: ChildrenHTML},
		{Prototype: `JSONRequest(str,str,map.str,str) str`, Object: JSONRequest},
		// For gentee
		{Prototype: `TempFile(str,str,str) str`, Object: TempFile},
		{Prototype: `obj(handle) obj`, Object: ObjHandle},
		{Prototype: `YamlToMap(str) map`, Object: YamlToMap},
		{Prototype: `YamlToObj(str) obj`, Object: YamlToObj},
		{Prototype: `CopyName(str) str`, Object: CopyName},
		{Prototype: `CloseLines(handle)`, Object: CloseLines},
		{Prototype: `GetLine(handle) str`, Object: GetLine},
		{Prototype: `ReadLines(str) handle`, Object: ReadLines},
		{Prototype: `ScanLines(handle) bool`, Object: ScanLines},
		{Prototype: `OpenCSV(str,str,str) handle`, Object: OpenCSV},
		{Prototype: `CloseCSV(handle)`, Object: CloseCSV},
		{Prototype: `ReadCSV(handle) bool`, Object: ReadCSV},
		{Prototype: `GetCSV(handle) obj`, Object: GetCSV},
	}
)

func Thread(rt *vm.Runtime, level int64) {
	if MainThread == nil {
		MainThread = rt
	}
	rt.Custom = &ThreadOptions{
		LogLevel: level,
		Refs:     []string{scriptTask.Header.Name},
	}
}

func PushRef(rt *vm.Runtime, ref string) {
	rt.Custom.(*ThreadOptions).Refs = append(rt.Custom.(*ThreadOptions).Refs, ref)
}

func PopRef(rt *vm.Runtime) error {
	refs := rt.Custom.(*ThreadOptions).Refs
	if len(refs) <= 1 {
		return fmt.Errorf(`empty refs`)
	}
	rt.Custom.(*ThreadOptions).Refs = refs[:len(refs)-1]
	return nil
}

func GetRef(rt *vm.Runtime) string {
	return strings.Join(rt.Custom.(*ThreadOptions).Refs, `/`)
}

func GetEonzaDynamic(name string) (ret string) {
	now := time.Now()
	switch name {
	case `date`:
		ret = now.Format(`20060102`)
	case `day`:
		ret = now.Format(`02`)
	case `month`:
		ret = now.Format(`01`)
	case `time`:
		ret = now.Format(`150405`)
	case `year`:
		ret = now.Format(`2006`)
	}
	return
}

func IsCond(rt *vm.Runtime, item *ConditionItem) (err error) {
	var (
		i              int64
		val, s, varVal string
	)
	if len(item.Var) == 0 && len(item.Value) == 0 {
		return fmt.Errorf(`empty variable in If Statement`)
	}
	if val, err = Macro(item.Value); err != nil {
		return
	}
	if len(item.Var) > 0 {
		if varVal, err = GetVar(item.Var); err != nil || (IsVar(item.Var) == 0 &&
			(strings.ContainsAny(item.Var, ` #[.`) || IsVarObj(item.Var) > 0)) {
			var found bool
			if varVal, found = ReplaceObj(item.Var); !found {
				if varVal, err = Macro(item.Var); err != nil {
					return
				}
			}
			err = nil
		}
	}
	switch item.Cmp {
	case `contains`:
		item.result = strings.Contains(varVal, val)
	case `equal`:
		if len(item.Value) == 0 {
			var i int64
			if len(varVal) > 0 && varVal != `0` && varVal != `false` {
				i = 1
			}
			item.result = i == 0
		} else {
			item.result = varVal == val
		}
	case `fileexists`:
		if len(item.Var) > 0 {
			s = varVal
		} else {
			s = val
		}
		if i, err = vm.ExistFile(rt, s); err != nil {
			return
		}
		item.result = i != 0
	case `envexists`:
		if len(item.Var) > 0 {
			s = varVal
		} else {
			s = val
		}
		_, item.result = os.LookupEnv(s)
	case `match`:
		i, err = vm.MatchºStrStr(varVal, val)
		item.result = i != 0
	case `starts`:
		item.result = strings.HasPrefix(varVal, val)
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

func GetConst(name string) (ret string, err error) {
	if len(name) == 0 {
		err = fmt.Errorf("invalid value")
		return
	}
	var ok bool
	ret, ok = (*dataScript.Global)[name]
	if ok && ret == EonzaDynamic {
		ret = GetEonzaDynamic(name)
	}
	return
}

func GetVar(name string) (ret string, err error) {
	if IsVar(name) != 0 {
		id := len(dataScript.Vars) - 1
		ret, err = Macro(dataScript.Vars[id][name])
	} else if strings.ContainsAny(name, `[.`) {
		var found bool
		if ret, found = ReplaceObj(name); !found {
			ret = ``
		}
	}
	return
}

func GetVarBytes(name string) (ret string, err error) {
	if IsVar(name) != 0 {
		ret = dataScript.Vars[len(dataScript.Vars)-1][name]
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

func GetVarRaw(name string) (ret string, err error) {
	if IsVar(name) != 0 {
		ret = dataScript.Vars[len(dataScript.Vars)-1][name]
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

func InitCmd(rt *vm.Runtime, logLevel int64, name string, pars ...interface{}) int64 {
	prevLevel := rt.Custom.(*ThreadOptions).LogLevel
	params := make([]string, len(pars))
	for i, par := range pars {
		val := fmt.Sprint(par)
		if len(val) > 64 {
			val = val[:64] + `...`
		}
		switch par.(type) {
		case string:
			params[i] = `"` + val + `"`
		default:
			params[i] = val
		}
	}
	if name != `source-code` {
		LogOutput(rt, LOG_INFO, fmt.Sprintf("=> %s(%s)", name, strings.Join(params, `, `)))
	}

	if logLevel != LOG_INHERIT {
		SetLogLevel(rt, logLevel)
	}
	return prevLevel
}

func IsEntry() int64 {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	if len(dataScript.Vars) == 1 {
		return 1
	}
	return 0
}

func IsVar(key string) int64 {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	_, ret := dataScript.Vars[len(dataScript.Vars)-1][key]
	if ret {
		return 1
	}
	return 0
}

func loadForm(data string, form *[]map[string]interface{}) (string, error) {

	var (
		dataList []map[string]interface{}
		ref      string
	)

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
					return ``, err
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
			if len(ref) < 12 {
				if id, err := strconv.ParseInt(item["type"].(string), 10, 32); err == nil &&
					id <= int64(PNumber) {
					ref += varname
				}
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
	return ref, nil
}

func Form(rt *vm.Runtime, data string) error {
	ch := make(chan bool)
	formList := make([]map[string]interface{}, 0, 32)

	ref, _ := loadForm(data, &formList)
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
		Ref:        GetRef(rt) + `:` + ref,
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

func LogOutput(rt *vm.Runtime, level int64, message string) {
	var mode = []string{``, `ERROR`, `WARN`, `FORM`, `INFO`, `DEBUG`}
	if level < LOG_ERROR || level > LOG_DEBUG {
		return
	}
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	if level > rt.Custom.(*ThreadOptions).LogLevel {
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
						if value == EonzaDynamic {
							value = GetEonzaDynamic(key[1:])
						}
					}
				} else if key[0] == '@' && scriptTask.Header.SecureConsts != nil {
					value, ok = scriptTask.Header.SecureConsts[key[1:]]
					if ok {
						result = append(result, []rune(value)...)
						clearName()
						isName = false
						continue
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

func SetLogLevel(rt *vm.Runtime, level int64) int64 {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	ret := rt.Custom.(*ThreadOptions).LogLevel
	if level >= LOG_DISABLE && level < LOG_INHERIT {
		rt.Custom.(*ThreadOptions).LogLevel = level
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
	if IsEntry() == 1 {
		return nil
	}
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

func InitData(chLogout chan string, chForm chan FormInfo, chReport chan Report, glob *map[string]string) {
	dataScript.Vars = make([]map[string]string, 0, 8)
	dataScript.chLogout = chLogout
	dataScript.chForm = chForm
	dataScript.chReport = chReport
	dataScript.Global = glob
}

func InitEngine(outerLib []gentee.EmbedItem) error {
	return gentee.Customize(&gentee.Custom{
		Embedded: append(customLib, outerLib...),
	})
}

func CopyClipboard(data string) error {
	return clipboard.WriteAll(data)
}

func GetClipboard() (string, error) {
	return clipboard.ReadAll()
}

func Unsupported(name string) error {
	return fmt.Errorf(`The '%s' function is unsupported`, name)
}

func LoadIni(buf *core.Buffer) (cfg *ini.File, err error) {
	cfg, err = ini.Load(buf.Data)
	return
}

func GetIniValue(cfg *ini.File, section, key, varname, defvalue string) (ret int64, err error) {
	sec := cfg.Section(section)
	if sec != nil {
		if sec.HasKey(key) {
			ret = 1
			err = SetVar(varname, sec.Key(key).String())
		}
	}
	if ret == 0 {
		if len(defvalue) == 0 {
			err = fmt.Errorf(`%s key doesn't exist in INI file`, key)
		} else {
			SetVar(varname, defvalue)
		}
	}
	return
}
