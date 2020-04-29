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
	ChStdin  chan []byte
	ChStdout chan []byte
	ChSystem chan int
}

func (script *Script) Run(options Settings) (interface{}, error) {
	var (
		settings   gentee.Settings
		rIn, wIn   *os.File
		rOut, wOut *os.File
	)
	settings.SysChan = options.ChSystem
	rIn, wIn, _ = os.Pipe()
	settings.Stdin = rIn
	rOut, wOut, _ = os.Pipe()
	settings.Stdout = wOut
	defer func() {
		wIn.Close()
		wOut.Close()
	}()

	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := rOut.Read(buf)
			buf = buf[:n]
			if err != nil {
				break
			}
			options.ChStdout <- buf
		}
	}()
	go func() {
		var buf []byte
		for {
			buf = <-options.ChStdin
			_, err := wIn.Write(buf)
			if err != nil {

			}
		}
	}()
	return script.Exec.Run(settings)
}
