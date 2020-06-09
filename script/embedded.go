// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"strings"
	"sync"
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
	LogLevel int64
	Mutex    sync.Mutex
	chLogout chan string
}

var (
	dataScript Data
	customLib  = []gentee.EmbedItem{
		{Prototype: `initcmd(str)`, Object: InitCmd},
		{Prototype: `LogOutput(int,str)`, Object: LogOutput},
		{Prototype: `SetLogLevel(int) int`, Object: SetLogLevel},
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
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	if level > dataScript.LogLevel {
		return
	}
	dataScript.chLogout <- fmt.Sprintf("[%s] %s %s",
		mode[level], time.Now().Format(`2006/01/02 15:04:05`), message)
}

func SetLogLevel(level int64) int64 {
	dataScript.Mutex.Lock()
	defer dataScript.Mutex.Unlock()
	ret := dataScript.LogLevel
	if level >= LOG_DISABLE && level < LOG_INHERIT {
		dataScript.LogLevel = level
	}
	return ret
}

func InitData(chLogout chan string) {
	dataScript.chLogout = chLogout
}

func InitEngine() error {
	return gentee.Customize(&gentee.Custom{
		Embedded: customLib,
	})
}
