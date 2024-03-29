// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/gob"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eonza/lib"
	"eonza/script"

	"github.com/gentee/gentee"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

var (
	stopchan    = make(chan os.Signal)
	scriptTask  *script.Script
	consoleData []byte
	isShutdown  bool
	outerLib    = []gentee.EmbedItem{
		{Prototype: `PkgFile(str,str)`, Object: PkgFile},
	}
)

func main() {
	var (
		e       *echo.Echo
		psw     string
		isRun   bool
		install bool
	)
	if isRun = CheckConsole(); isRun && len(consoleData) == 0 {
		return
	}
	golog.SetTimeFormat("2006/01/02 15:04:05")
	flag.StringVar(&cfg.path, "cfg", "", "The path of the `config file`")
	flag.StringVar(&psw, "psw", "", "The login password")
	flag.BoolVar(&install, "install", false, "only install")
	flag.Parse()
	if err := script.InitEngine(outerLib); err != nil {
		golog.Fatal(err)
	}
	script.InitWorkspace()
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})

	if !isRun {
		fi, err := os.Stdin.Stat()
		if err != nil {
			golog.Fatal(err)
		}
		isRun = fi.Mode()&os.ModeNamedPipe != 0
	}
	LoadAssets(isRun)
	if isRun {
		var err error
		IsScript = true
		scriptTask, err = script.Decode(consoleData)
		if err != nil {
			golog.Fatal(err)
		}
		if err = LoadCustomAsset(scriptTask.Header.AssetsDir, scriptTask.Header.HTTP.Theme); err != nil {
			golog.Fatal(err)
		}
		e = RunServer(*scriptTask.Header.HTTP)
		go func() {
			start := time.Now()
			settings := initTask()
			setStatus(TaskActive)
			_, err := scriptTask.Run(settings)
			if script.IsTimeout {
				time.Sleep(time.Until(script.Timeout))
			}
			if err == nil {
				setStatus(TaskFinished)
			} else if err.Error() == `code execution has been terminated` {
				// TODO: added special func or compare errID
				setStatus(TaskTerminated)
			} else {
				setStatus(TaskFailed, err)
			}
			<-chFinish
			if scriptTask.Header.HTTP.Open || scriptTask.Header.HTTP.Host != Localhost {
				if duration := time.Since(start).Milliseconds(); duration < TimeoutOpen {
					time.Sleep(time.Duration(TimeoutOpen-duration) * time.Millisecond)
				}
			}
			closeTask()
			stopchan <- os.Kill
		}()
	} else {
		LoadConfig()
		LoadStorage(psw)
		if install {
			return
		}
		hideConsole()
		ProInit(storage.Settings.PasswordHash, uint32(storage.PassCounter))
		LoadUsersSettings()
		defer CloseLog()
		if err := LoadCustomAsset(cfg.AssetsDir, cfg.HTTP.Theme); err != nil {
			golog.Fatal(err)
		}
		LoadNotifications()
		InitScripts()
		CreateSysTray()
		RunCron()
		e = RunServer(cfg.HTTP)
	}
	signal.Notify(stopchan, os.Kill, os.Interrupt, syscall.SIGTERM)
	sig := <-stopchan
	if !IsScript {
		CloseTaskManager()
	} else if sig != os.Kill && task.Status < TaskFinished {
		lib.LocalPost(scriptTask.Header.ServerPort, `api/taskstatus`,
			TaskStatus{
				TaskID: task.ID,
				Status: TaskTerminated,
				Time:   time.Now().Unix(),
			})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	isShutdown = true
	e.Shutdown(ctx)
}
