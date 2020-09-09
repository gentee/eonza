// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentee/gentee/core"
)

func request(url string) (string, error) {
	var (
		res *http.Response
		err error
	)
	buf := core.NewBuffer()
	res, err = http.Get(`http://localhost:` + url)
	if err == nil {
		buf.Data, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
	}
	return string(buf.Data), err
}

func IsConsole() bool {
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
		if answer != Success {
			if port = os.Getenv(`EZPORT`); len(port) > 0 {
				answer, _ = request(fmt.Sprintf("%s/ping", port))
			}
		}
		if answer != Success {
			return output("eonza has not been run or listens to custom port")
		}
		/*		answer, err = request(fmt.Sprintf("%s/api/run?name=%s&silent=true&console=true",
					port, os.Args[1]))
				if err != nil {
					return output(err)
				}
				var response Response
				if err = json.Unmarshal([]byte(answer), &response); err != nil {
					return output(err)
				}
				if !response.Success {
					return output(response.Error)
				}*/
		return true
	}
	return false
}
