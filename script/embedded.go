// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"strings"
	"time"

	"github.com/gentee/gentee"
)

const (
	LOG_DISABLE = iota
	LOG_ERROR
	LOG_WARN
	LOG_INFO
	LOG_DEBUG
	LOG_INHERIT
)

type Data struct {
	chLogout chan string
}

var (
	dataScript Data
	customLib  = []gentee.EmbedItem{
		{Prototype: `LogOutput(int,str)`, Object: LogOutput},
		{Prototype: `initcmd(str)`, Object: InitCmd},
	}
)

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
	LogOutput(LOG_DEBUG, fmt.Sprintf("=> %s(%s)", name, strings.Join(params, `, `)))
	return true
}

func LogOutput(level int64, message string) {
	var mode = []string{``, `ERROR`, `WARN`, `INFO`, `DEBUG`}
	if level < LOG_ERROR || level > LOG_DEBUG {
		return
	}
	dataScript.chLogout <- fmt.Sprintf("[%s] %s %s",
		mode[level], time.Now().Format(`2006/01/02 15:04:05`), message)
}

func InitData(chLogout chan string) {
	dataScript.chLogout = chLogout
}

func InitEngine() error {
	return gentee.Customize(&gentee.Custom{
		Embedded: customLib,
	})
}
