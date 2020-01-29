// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/golog"
)

const (
	logModeFile   = `file`
	logModeStdout = `stdout`

	logLevelDisable = `disable`
	logLevelError   = `error`
	logLevelWarning = `warn`
	logLevelInfo    = `info`
)

var (
	logFile *os.File
)

// SetLogging sets the options of the logging
func SetLogging(basename string) {
	if len(cfg.Log.Level) == 0 {
		cfg.Log.Level = logLevelInfo
	}
	if strings.Index(cfg.Log.Mode, logModeStdout) < 0 {
		golog.SetOutput(ioutil.Discard)
	}
	if cfg.Log.Level != logLevelDisable && strings.Index(cfg.Log.Mode, logModeFile) >= 0 {
		logFile, err := os.OpenFile(filepath.Join(defDir(cfg.Log.Dir), basename+`.log`),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			golog.Fatal(err)
		}
		golog.AddOutput(logFile)
	}
	golog.SetLevel(cfg.Log.Level)
	golog.Info(`Start`)

}

// CloseLog close file handle
func CloseLog() {
	if logFile != nil {
		logFile.Close()
	}
}
