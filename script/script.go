// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"github.com/gentee/gentee"
)

type Script struct {
	Header Header       // script header
	Exec   *gentee.Exec // Bytecode
}

var (
//	scripts = make(map[string]*Script)
)

func (script *Script) Run() (interface{}, error) {
	return script.Exec.Run(gentee.Settings{})
}
