// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import "eonza/lib"

const (
	// Version of the application
	Version = "1.28.0"
	// DefPort is the default web-server port
	DefPort = 3234
	// DefTheme is the default web-server theme
	DefTheme = `default`
	// DefAssets is the default name of assets directory
	DefAssets = `assets`
	// DefPackages is the default name of package directory
	DefPackages = `packages`
	// DefLogs is the default name of log directory
	DefLog = `log`
	// DefUsers is the default name of users directory
	DefUsers = `users`
	// UserExt contains the user's settings
	UserExt = `.usr`
	// Success is the success ping answer
	Success      = `ok`
	HistoryLimit = 7
	RunLimit     = 20
	// Number of reserved ports
	PortsPool    = 1000
	TimeFormat   = `2006/01/02 15:04:05`
	TimeoutOpen  = 2000
	SourceCode   = `source-code`
	Function     = `function`
	CallFunction = `call-function`
	Return       = `return.eonza`
	DefLang      = 0
	// ConsolePrefix is the prefix of eonza console version
	ConsolePrefix = `ez`
	Localhost     = lib.Localhost
	// DefTaskLimit is the maximum running scripts in playground mode
	DefTaskLimit = 2
)

// AppInfo contains information about the application
type AppInfo struct {
	Title     string
	Copyright string
	Homepage  string
	Email     string
	Lang      string
	Issue     string
}

var (
	VerType     string
	CompileDate string
)

func GetVersion() string {
	ret := Version
	if VerType == `beta` {
		ret += `b`
	}
	return ret
}
