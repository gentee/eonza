// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"time"

	"github.com/gentee/gentee"
)

const (
	LOG_DISABLE = iota
	LOG_ERROR
	LOG_WARN
	LOG_INFO
	LOG_DEBUG
)

type Data struct {
	chLogout chan string
}

var (
	dataScript Data
	customLib  = []gentee.EmbedItem{
		{Prototype: `LogOutput(int,str)`, Object: LogOutput},
	}
)

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
