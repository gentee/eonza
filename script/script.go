// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"os"

	"github.com/gentee/gentee"
)

type Script struct {
	Header Header       // script header
	Exec   *gentee.Exec // Bytecode
}

type Settings struct {
	ChStdin        chan []byte
	ChStdout       chan []byte
	ChSystem       chan int
	ProgressHandle gentee.ProgressFunc
}

var scriptTask *Script

func (script *Script) Run(options Settings) (interface{}, error) {
	var (
		settings   gentee.Settings
		rIn, wIn   *os.File
		rOut, wOut *os.File
		conOut     *os.File
	)
	settings.SysChan = options.ChSystem
	rOut, wOut, _ = os.Pipe()
	settings.Stdout = wOut
	settings.Stderr = wOut
	defer func() {
		wOut.Close()
	}()
	if script.Header.Console {
		conOut = os.Stdout
	} else {
		rIn, wIn, _ = os.Pipe()
		settings.Stdin = rIn
		defer func() {
			wIn.Close()
		}()
		go func() {
			var buf []byte
			for {
				buf = <-options.ChStdin
				_, err := wIn.Write(buf)
				if err != nil {
					break
				}
			}
		}()
	}
	go func(con bool) {
		for {
			buf := make([]byte, 1024)
			n, err := rOut.Read(buf)
			buf = buf[:n]
			if err != nil {
				break
			}
			if con {
				conOut.Write(buf)
			}
			options.ChStdout <- buf
		}
	}(script.Header.Console)
	if script.Header.IsPlayground {
		settings.IsPlayground = true
		settings.Playground.Path = script.Header.Playground.Dir
		settings.Playground.AllSizeLimit = script.Header.Playground.Summary
		settings.Playground.FilesLimit = int(script.Header.Playground.Files)
		settings.Playground.SizeLimit = script.Header.Playground.Size
	}
	settings.ProgressHandle = options.ProgressHandle
	scriptTask = script
	return script.Exec.Run(settings)
}
