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

type Package struct {
	Exec *gentee.Exec // Bytecode
}

func Send() error {
	var (
		data bytes.Buffer
	)
	workspace := gentee.New()
	bcode, _, err := workspace.Compile(`run {
		Println("Alright")
		Open("http://google.com")
	}`, "hello")
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(&data)
	err = enc.Encode(Package{
		Exec: bcode,
	})
	if err != nil {
		return err
	}
	command := exec.Command(lib.AppPath())
	command.Stdin = &data
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	fmt.Println(`START`)
	return command.Start()
}

func Run() (err error) {
	var (
		data bytes.Buffer
		pkg  Package
	)
	if _, err = data.ReadFrom(os.Stdin); err != nil {
		return nil
	}
	dec := gob.NewDecoder(&data)

	if err = dec.Decode(&pkg); err == nil {
		_, err = pkg.Exec.Run(gentee.Settings{})
	}
	return
}
