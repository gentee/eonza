// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"time"

	"eonza/script"

	"github.com/kataras/golog"
)

func main() {
	golog.SetTimeFormat("2006/01/02 15:04:05")
	flag.StringVar(&cfg.path, "cfg", "", "The path of the `config file`")
	flag.Parse()
	script.InitWorkspace()

	fi, err := os.Stdin.Stat()
	if err != nil {
		golog.Fatal(err)
	}
	if fi.Mode()&os.ModeNamedPipe != 0 {
		IsScript = true
		scriptTask, err := script.Decode()
		if err != nil {
			golog.Fatal(err)
		}
		if err = LoadCustomAsset(scriptTask.Header.AssetsDir, scriptTask.Header.HTTP.Theme); err != nil {
			golog.Fatal(err)
		}
		RunServer(WebSettings{
			Port: scriptTask.Header.HTTP.Port,
			Open: true,
			Lang: scriptTask.Header.Lang,
		})
		scriptTask.Run()
		time.Sleep(2000 * time.Second)
		return
	}

	LoadConfig()
	LoadStorage()
	defer CloseLog()
	if err = LoadCustomAsset(cfg.AssetsDir, cfg.HTTP.Theme); err != nil {
		golog.Fatal(err)
	}
	RunServer(WebSettings{
		Port: cfg.HTTP.Port,
		Open: cfg.HTTP.Open,
		Lang: storage.Settings.Lang,
	})
}
