// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"regexp"

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

// FindFirstRegExpºStrStr returns an array of the first successive matches of the expression
func FindFirstRegExpºStrStr(src, rePattern string) (*core.Array, error) {
	re, err := regexp.Compile(rePattern)
	if err != nil {
		return nil, err
	}
	list := re.FindStringSubmatch(src)
	out := core.NewArray()
	for _, sub := range list {
		out.Data = append(out.Data, sub)
	}
	return out, nil
}
