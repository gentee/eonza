// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
)

func MapCondition(rt *vm.Runtime, item *core.Map) (int64, error) {
	var (
		cond ConditionItem
		ok   bool
		tmp  interface{}
	)

	if tmp, ok = item.Data["var"]; ok {
		cond.Var = fmt.Sprint(tmp.(*core.Obj).Data)
	}
	if tmp, ok = item.Data["cmp"]; ok {
		cond.Cmp = fmt.Sprint(tmp.(*core.Obj).Data)
	}
	if tmp, ok = item.Data["value"]; ok {
		cond.Value = fmt.Sprint(tmp.(*core.Obj).Data)
	}
	if tmp, ok = item.Data["not"]; ok {
		not := fmt.Sprint(tmp.(*core.Obj).Data)
		cond.Not = !(len(not) == 0 || not == `0` || not == `false`)
	}
	if err := IsCond(rt, &cond); err != nil {
		return 0, err
	}
	if cond.result {
		return 1, nil
	}
	return 0, nil
}

func ObjToStr(key string) (string, bool) {
	if len(dataScript.ObjVars) > 0 {
		if v, ok := dataScript.ObjVars[len(dataScript.ObjVars)-1].Load(key); ok {
			switch vm.Type(v.(*core.Obj)) {
			case `int`, `float`, `str`, `bool`:
				return fmt.Sprint(v.(*core.Obj).Data), true
			}
		}
	}
	return ``, false
}

func ReplaceObj(key string) (ret string, found bool) {
	var (
		obj    *core.Obj
		off    int
		aindex int64
		v      interface{}
		index  int
	)
	iMap := len(dataScript.ObjVars) - 1
	input := []rune(key)
	getObj := func(i int) bool {
		var ok bool
		name := string(input[off:i])
		if len(name) > 0 {
			if obj == nil {
				if v, ok = dataScript.ObjVars[iMap].Load(name); ok {
					obj = v.(*core.Obj)
				} else {
					return false
				}
			} else if obj, _ = vm.ItemºObjStr(obj, name); obj == nil {
				return false
			}
		}
		off = i + 1
		return true
	}
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '.':
			if index == 0 && !getObj(i) {
				return
			}
		case '[':
			if index == 0 && !getObj(i) {
				return
			}
			index++
		case ']':
			index--
			if index == 0 {
				if ind, err := macro(string(input[off:i])); err != nil {
					return
				} else {
					switch obj.Data.(type) {
					case *core.Map:
						if obj, _ = vm.ItemºObjStr(obj, ind); obj == nil {
							return
						}
						off = i + 1
					case *core.Array:
						if aindex, err = strconv.ParseInt(ind, 10, 64); err != nil {
							return
						}
						if obj, _ = vm.ItemºObjInt(obj, aindex); obj == nil {
							return
						}
						off = i + 1
					default:
						return
					}
				}
			}
		}
	}
	if off < len(input) && !getObj(len(input)) {
		return
	}
	if obj != nil {
		switch obj.Data.(type) {
		case *core.Array, *core.Map:
		default:
			ret = fmt.Sprint(obj.Data)
			found = true
		}
	}
	return
}

func GetVarObj(name string) (*core.Obj, error) {
	val, ok := dataScript.ObjVars[len(dataScript.ObjVars)-1].Load(name)
	if !ok {
		subfields := strings.Split(name, `.`)
		if len(subfields) > 1 {
			if val, ok = dataScript.ObjVars[len(dataScript.ObjVars)-1].Load(subfields[0]); ok {
				ret := val.(*core.Obj)
				for i := 1; i < len(subfields); i++ {
					if ret, _ = vm.ItemºObjStr(ret, subfields[i]); ret == nil {
						break
					}
				}
				if ret != nil {
					return ret, nil
				}
			}
		}
		return nil, fmt.Errorf(`object var %s doesn't exist`, name)
	}
	return val.(*core.Obj), nil
}

func IsVarObj(name string) int64 {
	_, ok := dataScript.ObjVars[len(dataScript.ObjVars)-1].Load(name)
	if ok {
		return 1
	}
	return 0
}

func setRawVarObj(shift int, name string, value *core.Obj) error {
	off := len(dataScript.ObjVars) - 1 - shift
	if off < 0 {
		return fmt.Errorf(`set shift obj var %s error`, name)
	}
	dataScript.ObjVars[off].Store(name, value)
	return nil
}

func ResultVarObj(name string, value *core.Obj) error {
	return setRawVarObj(1, name, value)
}

func SetVarObj(name string, value *core.Obj) error {
	return setRawVarObj(0, name, value)
}
