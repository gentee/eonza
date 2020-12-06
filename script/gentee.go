// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"

	"github.com/gentee/gentee"
	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
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

// Subbuf(buf, int, int) buf
func Subbuf(buf *core.Buffer, off int64, size int64) (*core.Buffer, error) {
	if off < 0 || off+size > int64(len(buf.Data)) {
		return nil, fmt.Errorf(vm.ErrorText(core.ErrInvalidParam))
	}
	ret := core.NewBuffer()
	ret.Data = append(ret.Data, buf.Data[off:off+size]...)
	return ret, nil
}
