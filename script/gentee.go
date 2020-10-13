// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"os"

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

// UnsetEnv unsets the environment variable
func UnsetEnv(rt *vm.Runtime, name string) error {
	if rt.Owner.Settings.IsPlayground {
		// restore in gentee ErrorText(ErrPlayEnv))
		return fmt.Errorf(`[Playground] setting the environment variable is disabled`)
	}
	return os.Unsetenv(name)
}
