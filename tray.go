// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build tray

package main

import (
	"eonza/lib"
	"fmt"
	"os"

	"github.com/getlantern/systray"
	"github.com/kataras/golog"
)

var isTray bool = true

func CreateSysTray() {
	if storage.Settings.HideTray /*|| cfg.HTTP.Access == AccessHost*/ {
		return
	}
	go systray.Run(TrayReady, TrayExit)
}

func HideTray() {
	systray.Quit()
}

func TrayExit() {
}

func TrayReady() {
	systray.SetIcon(WebAsset(`favicon.ico`))
	title := storage.Settings.Title
	if len(title) == 0 {
		title = fmt.Sprintf("%s:%d", appInfo.Title, cfg.HTTP.Port)
	}
	systray.SetTitle(title)
	systray.SetTooltip(appInfo.Title)
	var langId int
	us := RootUserSettings()
	if len(us.Lang) > 0 {
		langId = langsId[us.Lang]
	}
	open := Lang(langId, `openbrowser`)
	mOpen := systray.AddMenuItem(open, open)
	systray.AddSeparator()
	shutdown := Lang(langId, `shutdown`)
	mQuit := systray.AddMenuItem(shutdown, shutdown)
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				lib.Open(fmt.Sprintf("http://%s:%d", Localhost, cfg.HTTP.Port))
			case <-mQuit.ClickedCh:
				golog.Info(`Tray shutdown`)
				stopchan <- os.Interrupt
				systray.Quit()
			}
		}
	}()
}
