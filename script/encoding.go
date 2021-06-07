// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/gob"
	"os"
	"os/exec"

	"eonza/lib"
	"eonza/users"

	"github.com/gentee/gentee"
)

type Header struct {
	Name         string
	Title        string
	AssetsDir    string
	LogDir       string
	Theme        string
	CDN          string
	Data         string
	Console      bool
	IsPlayground bool
	SourceCode   []byte
	Constants    map[string]string
	SecureConsts map[string]string
	Lang         string
	User         users.User
	Role         users.Role
	ClaimKey     string
	IsPro        bool
	IP           string
	TaskID       uint32
	ServerPort   int
	URLPort      int
	HTTP         *lib.HTTPConfig
	Playground   *lib.PlaygroundConfig
}

func Encode(header Header, source string) (*bytes.Buffer, error) {
	var (
		data bytes.Buffer
	)

	workspace := gentee.New()
	bcode, _, err := workspace.Compile(source, header.Name)
	if err != nil {
		return nil, err
	}
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(header); err != nil {
		return nil, err
	}
	if err = enc.Encode(bcode); err != nil {
		return nil, err
	}
	if header.Console {
		return &data, nil
	}
	command := exec.Command(lib.AppPath())
	command.Stdin = &data
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err = command.Start()
	go func() {
		if err == nil {
			_ = command.Wait()
		}
	}()
	return nil, err
}

func Decode(scriptData []byte) (script *Script, err error) {
	script = &Script{}
	data := bytes.NewBuffer(scriptData)
	if scriptData == nil {
		if _, err = data.ReadFrom(os.Stdin); err != nil {
			return
		}
	}
	dec := gob.NewDecoder(data)

	if err = dec.Decode(&script.Header); err == nil {
		err = dec.Decode(&script.Exec)
	}
	return
}

func ReplaceVars(input string, values map[string]string, glob *map[string]string) string {
	if len(values) == 0 && len(*glob) == 0 {
		return input
	}
	stack := make([]string, 0)
	ret, err := replace(values, []rune(input), &stack, glob)
	if err != nil {
		return input
	}
	return string(ret)
}
