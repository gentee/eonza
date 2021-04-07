// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentee/gentee/core"
)

func request(url string) ([]byte, error) {
	var (
		res *http.Response
		err error
	)
	buf := core.NewBuffer()
	res, err = http.Get(fmt.Sprintf(`http://%s:%s`, Localhost, url))
	if err == nil {
		buf.Data, err = io.ReadAll(res.Body)
		res.Body.Close()
	}
	return buf.Data, err
}

func CheckConsole() bool {
	port := fmt.Sprint(DefPort)
	appname, err := os.Executable()
	if err != nil {
		fmt.Println(err)
	}
	output := func(msg interface{}) bool {
		fmt.Println(msg)
		return true
	}
	appname = filepath.Base(appname)
	if strings.HasPrefix(appname, ConsolePrefix) {
		if len(os.Args) == 1 {
			fmt.Printf("Usage: %s <scriptname>\r\n", appname)
			return true
		}
		answer, _ := request(fmt.Sprintf("%s/ping", port))
		if string(answer) != Success {
			if port = os.Getenv(`EZPORT`); len(port) > 0 {
				answer, _ = request(fmt.Sprintf("%s/ping", port))
			}
		}
		if string(answer) != Success {
			return output("eonza has not been run or listens to custom port")
		}
		answer, err = request(fmt.Sprintf("%s/api/run?name=%s&silent=true&console=true",
			port, os.Args[1]))
		if err != nil {
			return output(err)
		}
		if answer[0] == '{' {
			var response Response
			if err = json.Unmarshal(answer, &response); err != nil {
				return output(err)
			}
			return output(response.Error)
		}
		consoleData = answer
		return true
	}
	return false
}
