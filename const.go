// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

const (
	// Version of the application
	Version = "1.0.0+1"
	// DefPort is the default web-server port
	DefPort = 3234
	// DefTheme is the default web-server theme
	DefTheme = `default`
	// DefAssets is the default name of assets directory
	DefAssets = `assets`
	// DefLogs is the default name of log directory
	DefLog = `log`
	// DefUsers is the default name of users directory
	DefUsers = `users`
	// UserExt contains the user's settings
	UserExt = `.usr`
	// Success is the success api answer
	Success = `ok`
	// HistoryLimit is the limit of history count
	HistoryLimit = 7
)

// AppInfo contains information about the application
type AppInfo struct {
	Title     string
	Copyright string
	Homepage  string
	Email     string
	Lang      string
	License   string
}
