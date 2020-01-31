// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"

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
		script.Run()
		return
	}

	LoadConfig()
	defer CloseLog()
	RunServer(WebSettings{
		Port: cfg.HTTP.Port,
		Open: cfg.HTTP.Open,
	})
}
