// Copyright 2022 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gentee/gentee/core"
	"github.com/labstack/echo/v4"
)

var (
	Pkgs   = make(map[string]int)
	chPkgs chan string
	muPkgs sync.Mutex
)

func CmdPkg(cmd string, obj *core.Obj) error {
	path := strings.SplitN(cmd, `/`, 2)
	if len(path) != 2 {
		return fmt.Errorf(`invalid command %s`, cmd)
	}
	pkg := path[0]

	var (
		port int
		ok   bool
		err  error
	)

	muPkgs.Lock()
	if port, ok = Pkgs[pkg]; !ok {
		app := filepath.Join(scriptTask.Header.PackagesDir, pkg)
		if _, err = os.Stat(app); os.IsNotExist(err) {
			muPkgs.Unlock()
			return fmt.Errorf(`can't find '%s' package`, pkg)
		}
		var out bytes.Buffer

		app = filepath.Join(app, pkg)
		cmd := exec.Command(app, fmt.Sprintf("-t=%d", scriptTask.Header.TaskID),
			fmt.Sprintf("-p=%d", scriptTask.Header.HTTP.LocalPort))
		cmd.Stderr = &out
		chPkgs = make(chan string)
		if err = cmd.Start(); err == nil {
			select {
			case v := <-chPkgs:
				if strings.HasPrefix(v, `#`) {
					port, err = strconv.Atoi(v[1:])
				} else {
					err = fmt.Errorf(v)
				}
			case <-time.After(300 * time.Millisecond):
				if out.Len() > 0 {
					err = fmt.Errorf(out.String())
				} else {
					err = fmt.Errorf(`no answer from %s`, app)
				}
			}
		}
		chPkgs = nil
		if err == nil && port > 0 {
			Pkgs[pkg] = port
		}
	}
	muPkgs.Unlock()
	if err != nil {
		return err
	}
	fmt.Println(`CMD`, path[1])
	return nil
}

func cmdStart(cmd *CmdData) CmdData {
	if v, ok := cmd.Value.(int); ok {
		chPkgs <- fmt.Sprintf(`#%d`, v)
	} else {
		chPkgs <- fmt.Sprintf(`invalid port %v`, cmd.Value)
	}
	return CmdData{}
}

func cmdHandle(c echo.Context) error {
	var response CmdData

	cmd, err := ProcessCmd(scriptTask.Header.TaskID, c.Request().Body)
	fmt.Println(`get cmd:`, cmd.Cmd)
	if err == nil {
		response = cmdStart(cmd)
	} else {
		response.Error = err.Error()
	}
	response.TaskID = scriptTask.Header.TaskID
	return c.Blob(http.StatusOK, "application/octet-stream", ResponseCmd(&response))
}

func CmdServer(e *echo.Echo) {
	e.POST(`/cmd`, cmdHandle)
}

func ClosePkgs() {

}
