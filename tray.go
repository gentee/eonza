// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build tray

package main

import (
	"eonza/lib"
	es "eonza/script"
	"fmt"
	"os"

	"github.com/gentee/systray"
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
	chanFav := make(chan *systray.MenuItem)

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
	glob := &langRes[langId]
	scriptTitle := func(name string) string {
		ret := name
		if iscript := getScript(name); iscript != nil {
			ret = es.ReplaceVars(iscript.Settings.Title, iscript.Langs[us.Lang], glob)
		}
		if len(ret) == 0 {
			return name
		}
		return ret
	}
	for i, item := range us.Favs {
		var m *systray.MenuItem
		if (item.IsFolder && len(item.Children) == 0) || i > 15 {
			continue
		}
		m = systray.AddMenuItemChan(scriptTitle(item.Name), item.Name, chanFav)
		if item.IsFolder {
			for j, sub := range item.Children {
				if j > 15 {
					break
				}
				m.AddSubMenuItemChan(scriptTitle(sub.Name), sub.Name, chanFav)
			}
		}
	}
	systray.AddSeparator()
	open := Lang(langId, `openbrowser`)
	mOpen := systray.AddMenuItem(open, open)
	systray.AddSeparator()
	shutdown := Lang(langId, `shutdown`)
	mQuit := systray.AddMenuItem(shutdown, shutdown)
	go func() {
		var menuItem *systray.MenuItem
		for {
			select {
			case menuItem = <-chanFav:
				_, name := menuItem.Name()
				if len(name) > 0 {
					_, err := request(fmt.Sprintf("%d/api/run?name=%s", cfg.HTTP.Port, name))
					if err != nil {
						golog.Error(err)
					}
				}
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
