// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"time"

	"github.com/gentee/gentee"
	"github.com/gentee/gentee/core"
	"gopkg.in/yaml.v2"
)

func YamlToMap(in string) (*core.Map, error) {
	var (
		tmp map[string]string
		ret interface{}
		err error
	)
	if err = yaml.Unmarshal([]byte(in), &tmp); err != nil {
		return nil, err
	}
	ret, err = gentee.Go2GenteeType(tmp, `map.str`)
	if err != nil {
		return nil, err
	}
	return ret.(*core.Map), nil
}

// replace to obj += obj
func AppendObj(obj *core.Obj, value *core.Obj) (*core.Obj, error) {
	var err error
	if obj == nil {
		return nil, fmt.Errorf(`obj nil`)
	}
	switch v := obj.Data.(type) {
	case *core.Array:
		v.Data = append(v.Data, value)
	default:
		err = fmt.Errorf(`wrong obj value`)
	}
	return obj, err
}

// dup
func objAny(val interface{}) *core.Obj {
	obj := core.NewObj()
	obj.Data = val
	return obj
}

// dup
func toTime(it *core.Struct) time.Time {
	utc := time.Local
	if it.Values[6].(int64) == 1 {
		utc = time.UTC
	}
	return time.Date(int(it.Values[0].(int64)), time.Month(it.Values[1].(int64)),
		int(it.Values[2].(int64)), int(it.Values[3].(int64)), int(it.Values[4].(int64)),
		int(it.Values[5].(int64)), 0, utc)
}

// + str(time)
func TimeToStr(it *core.Struct) string {
	return toTime(it).Format(`2006-01-02 15:04:05`)
}

// + obj(finfo)
func FinfoToObj(finfo *core.Struct) *core.Obj {
	obj := core.NewObj()
	val := core.NewMap()
	val.SetIndex(`Name`, objAny(finfo.Values[0]))
	val.SetIndex(`Size`, objAny(finfo.Values[1]))
	val.SetIndex(`Mode`, objAny(finfo.Values[2]))
	val.SetIndex(`Time`, TimeToStr(finfo.Values[3].(*core.Struct)))
	val.SetIndex(`IsDir`, objAny(finfo.Values[4].(int64) != 0))
	val.SetIndex(`Dir`, objAny(finfo.Values[5]))
	obj.Data = val
	return obj
}

//=========== Renamed

// itemºObjStr
func ItemºObjStr(val *core.Obj, key string) (ret *core.Obj, err error) {
	if val == nil || val.Data == nil {
		return
	}
	switch v := val.Data.(type) {
	case *core.Map:
		if item, ok := v.Data[key]; ok {
			ret = item.(*core.Obj)
		}
	default:
		err = fmt.Errorf(`wrong obj type`)
	}
	return
}

// itemºObjInt
func ItemºObjInt(val *core.Obj, ind int64) (ret *core.Obj, err error) {
	if val == nil || val.Data == nil {
		return
	}
	switch v := val.Data.(type) {
	case *core.Array:
		if ind >= 0 && ind < int64(len(v.Data)) {
			ret = v.Data[ind].(*core.Obj)
		}
	default:
		err = fmt.Errorf(`obj is not array`)
	}
	return
}
