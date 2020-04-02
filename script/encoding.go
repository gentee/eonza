// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"

	"eonza/lib"

	"github.com/gentee/gentee"
)

type Header struct {
	Name       string
	AssetsDir  string
	Theme      string
	Lang       string
	UserID     uint32
	TaskID     uint32
	ServerPort int
	HTTP       *lib.HTTPConfig
}

func Encode(header Header) error {
	var (
		data bytes.Buffer
	)
	workspace := gentee.New()
	bcode, _, err := workspace.Compile(`run {
		Println("Alright")
//		Open("http://google.com")
	}`, "hello")
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(header); err != nil {
		return err
	}
	if err = enc.Encode(bcode); err != nil {
		return err
	}
	command := exec.Command(lib.AppPath())
	command.Stdin = &data
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	fmt.Println(`START`)
	return command.Start()
}

func Decode() (script *Script, err error) {
	var (
		data bytes.Buffer
	)
	script = &Script{}
	if _, err = data.ReadFrom(os.Stdin); err != nil {
		return
	}
	dec := gob.NewDecoder(&data)

	if err = dec.Decode(&script.Header); err == nil {
		err = dec.Decode(&script.Exec)
	}
	return
}
