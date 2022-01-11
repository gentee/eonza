// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
			if iv, ok := v.(*core.Obj).Data.(*goquery.Selection); ok {
				return iv.Text(), true
			}
			switch vm.Type(v.(*core.Obj)) {
			case `int`, `float`, `str`, `bool`:
				return fmt.Sprint(v.(*core.Obj).Data), true
			case `arr.obj`, `map.obj`:
				if ret, err := vm.Json(v.(*core.Obj)); err == nil {
					return ret, true
				}
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
			} else {
				if q, ok := obj.Data.(*goquery.Selection); ok {
					obj = core.NewObj()
					obj.Data, _ = q.Attr(name)
				} else if obj, _ = vm.ItemºObjStr(obj, name); obj == nil {
					return false
				}
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
		switch v := obj.Data.(type) {
		case *goquery.Selection:
			ret = v.Text()
			found = true
		case *core.Array, *core.Map:
			if jsonret, err := vm.Json(obj); err == nil {
				ret = jsonret
				found = true
			}
		default:
			ret = fmt.Sprint(v)
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
		return nil, fmt.Errorf(`variable object "%s" doesn't exist`, name)
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

func getRawVarObj(shift int, name string) (*core.Obj, error) {
	off := len(dataScript.ObjVars) - 1 - shift
	if off < 0 {
		return nil, fmt.Errorf(`set shift obj var %s error`, name)
	}
	ret, ok := dataScript.ObjVars[off].Load(name)
	if !ok {
		return nil, fmt.Errorf(`object variable %s doesn't exist`, name)
	}
	return ret.(*core.Obj), nil
}

func ResultVarObj(name string, value *core.Obj) error {
	return setRawVarObj(1, name, value)
}

func GetResultVarObj(name string) (*core.Obj, error) {
	return getRawVarObj(1, name)
}

func SetVarObj(name string, value *core.Obj) error {
	return setRawVarObj(0, name, value)
}

func SetJsonVar(name, input string) error {
	obj, err := vm.JsonToObj(input)
	if err != nil {
		return err
	}
	return setRawVarObj(0, name, obj)
}

func AppendToArray(name, value string) error {
	obj, err := GetVarObj(name)
	if err != nil {
		if strings.ContainsAny(name, `[]`) || len(name) == 0 {
			return err
		}
		names := strings.Split(name, `.`)
		arr := core.NewArray()
		obj = core.NewObj()
		obj.Data = arr
		if len(names) == 1 {
			SetVarObj(name, obj)
		} else {
			ownerName := strings.Join(names[:len(names)-1], `.`)
			owner, err := GetVarObj(ownerName)
			if err != nil {
				return err
			}
			imap, ok := owner.Data.(*core.Map)
			if !ok {
				return fmt.Errorf(`%s is not map object`, ownerName)
			}
			imap.SetIndex(names[len(names)-1], obj)
		}
	}
	if vm.IsArrayºObj(obj) == 0 {
		return fmt.Errorf(`%s is not array object`, name)
	}
	var val *core.Obj
	if val, err = GetVarObj(value); err != nil {
		val = core.NewObj()
		val.Data = value
	}
	obj.Data.(*core.Array).Data = append(obj.Data.(*core.Array).Data, val)
	return nil
}

func AppendToMap(name, key, value string) error {
	obj, err := GetVarObj(name)
	if err != nil {
		if strings.ContainsAny(name, `[]`) || len(name) == 0 {
			return err
		}
		names := strings.Split(name, `.`)
		amap := core.NewMap()
		obj = core.NewObj()
		obj.Data = amap
		if len(names) == 1 {
			SetVarObj(name, obj)
		} else {
			ownerName := strings.Join(names[:len(names)-1], `.`)
			owner, err := GetVarObj(ownerName)
			if err != nil {
				return err
			}
			imap, ok := owner.Data.(*core.Map)
			if !ok {
				return fmt.Errorf(`%s is not map object`, ownerName)
			}
			imap.SetIndex(names[len(names)-1], obj)
		}
	}
	if vm.IsMapºObj(obj) == 0 {
		return fmt.Errorf(`%s is not map object`, name)
	}
	var val *core.Obj
	if val, err = GetVarObj(value); err != nil {
		val = core.NewObj()
		val.Data = value
	}
	obj.Data.(*core.Map).SetIndex(key, val)
	return nil
}
