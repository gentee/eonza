// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"

	"github.com/kataras/golog"
)

/*
func autoDel() {
	if _, err := os.Stat(cfg.WebDir); err != nil {
		return
	}
	golog.Info(`Deleting static files and directories...`)
	RemoveCache()
}

func createDir(content *Content, path string, static bool) {
	dir := filepath.Join(path, content.dir)
	os.MkdirAll(dir, os.ModePerm)
	if static {
		for _, page := range content.pages {
			data, err := RenderPage(page.url)
			if err != nil {
				golog.Fatal(err)
			}
			if err = ioutil.WriteFile(filepath.Join(cfg.WebDir, filepath.FromSlash(page.url)),
				[]byte(data), os.ModePerm); err != nil {
				golog.Fatal(err)
			}
		}
	}
	for _, child := range content.children {
		createDir(child, dir, static)
	}
}

func createStatic() {
	golog.Info(`Creating static files and directories...`)
	cfg.mode = ModeDynamic
	createDir(contents, cfg.WebDir, true)
	cfg.mode = ModeStatic
}
*/
func main() {
	golog.SetTimeFormat("2006/01/02 15:04:05")
	flag.StringVar(&cfg.path, "cfg", "", "The path of the `config file`")
	flag.Parse()
	LoadConfig()
	fmt.Println(`OK`)
	/*	var err error

		LoadSettings()
		golog.Infof(`Mode: %s`, cfg.Mode)
		LoadContent()
		LoadTemplates()
		if cfg.AutoDel {
			autoDel()
		}
		if cfg.mode != ModeDynamic {
			os.MkdirAll(cfg.WebDir, os.ModePerm)
			if cfg.mode == ModeStatic {
				createStatic()
			} else {
				createDir(contents, cfg.WebDir, false)
			}
		}
		if cfg.mode == ModeLive {
			watcher, err = fsnotify.NewWatcher()
			if err != nil {
				golog.Fatal(err)
			}
			defer watcher.Close()
			if err = WatchDirs(); err != nil {
				golog.Fatal(err)
			}
			go StartLive()
		}
		RunServer()*/
}
