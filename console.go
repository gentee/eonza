// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"eonza/lib"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func CheckConsole() bool {
	port := DefPort
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
		if !pingHost(port) {
			if ezport := os.Getenv(`EZPORT`); len(ezport) > 0 {
				if uport, err := strconv.ParseUint(ezport, 10, 32); err == nil {
					port = int(uport)
					if !pingHost(port) {
						return output("eonza has not been run or listens to custom port")
					}
				}
			}
		}
		answer, err := lib.LocalGet(port, fmt.Sprintf("api/run?name=%s&silent=true&console=true",
			os.Args[1]))
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
