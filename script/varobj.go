// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/gentee/gentee/core"
)

var (
	mapObj sync.Map
)

func ReplaceObj(key string) (ret string, found bool) {
	var (
		obj    *core.Obj
		off    int
		aindex int64
		v      interface{}
		index  int
	)
	input := []rune(key)
	getObj := func(i int) bool {
		var ok bool
		name := string(input[off:i])
		if obj == nil {
			if v, ok = mapObj.Load(name); ok {
				obj = v.(*core.Obj)
			} else {
				return false
			}
		} else if obj, _ = ItemºObjStr(obj, name); obj == nil {
			return false
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
				if ind, err := Macro(string(input[off:i])); err != nil {
					return
				} else {
					switch obj.Data.(type) {
					case *core.Map:
						if obj, _ = ItemºObjStr(obj, ind); obj == nil {
							return
						}
					case *core.Array:
						if aindex, err = strconv.ParseInt(ind, 10, 64); err != nil {
							return
						}
						if obj, _ = ItemºObjInt(obj, aindex); obj == nil {
							return
						}
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

func SetVarObj(name string, value *core.Obj) {
	mapObj.Store(name, value)
}
