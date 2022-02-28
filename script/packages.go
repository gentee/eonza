// Copyright 2022 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
	"github.com/labstack/echo/v4"
)

type PkgHandle struct {
	Obj      *core.Obj
	Finished bool
	Unique   uint32
}

var (
	Pkgs       = make(map[string]int)
	chPkgs     chan string
	muPkgs     sync.Mutex
	ResultPkgs = make(map[uint32]chan CmdData)
)

func CmdPkg(cmd string, obj *core.Obj) (*PkgHandle, error) {
	path := strings.SplitN(cmd, `/`, 2)
	if len(path) != 2 {
		return nil, fmt.Errorf(`invalid command %s`, cmd)
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
			return nil, fmt.Errorf(`can't find '%s' package`, pkg)
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
		return nil, err
	}
	var val interface{}
	if obj != nil {
		val = ObjToIface(obj)
	}
	unique := rand.Uint32()
	ResultPkgs[unique] = make(chan CmdData, 1)
	_, err = SendCmd(port, &CmdData{
		Cmd:    path[1],
		TaskID: scriptTask.Header.TaskID,
		Unique: unique,
		Value:  val,
	})
	if err != nil {
		return nil, err
	}
	resp := <-ResultPkgs[unique]
	if resp.Finished {
		close(ResultPkgs[unique])
		delete(ResultPkgs, unique)
	}
	if len(resp.Error) != 0 {
		return nil, fmt.Errorf(resp.Error)
	}
	handle := PkgHandle{
		Unique:   unique,
		Finished: resp.Finished,
	}
	if resp.Value != nil {
		obj, err := vm.IfaceToObj(resp.Value)
		if err != nil {
			return nil, err
		}
		handle.Obj = obj
	}

	return &handle, nil
}

func CmdValue(handle *PkgHandle) *core.Obj {
	return handle.Obj
}

func CmdFinished(handle *PkgHandle) int64 {
	if handle.Finished {
		return 1
	}
	return 0
}

func cmdStart(cmd *CmdData) CmdData {
	if v, ok := cmd.Value.(int); ok {
		chPkgs <- fmt.Sprintf(`#%d`, v)
	} else {
		chPkgs <- fmt.Sprintf(`invalid port %v`, cmd.Value)
	}
	return CmdData{}
}

func cmdPing(cmd *CmdData) CmdData {
	return CmdData{}
}

func cmdHandle(c echo.Context) error {
	var response CmdData

	cmd, err := ProcessCmd(scriptTask.Header.TaskID, c.Request().Body)
	if err == nil {
		switch cmd.Cmd {
		case CmdStart:
			response = cmdStart(cmd)
		case CmdPing:
			response = cmdPing(cmd)
		default:
			response.Error = fmt.Sprintf(`unknown command %s`, cmd.Cmd)
		}
	} else {
		response.Error = err.Error()
	}
	response.TaskID = scriptTask.Header.TaskID
	return c.Blob(http.StatusOK, "application/octet-stream", ResponseCmd(&response))
}

func cmdResultHandle(c echo.Context) error {
	var response CmdData

	cmd, err := ProcessCmd(scriptTask.Header.TaskID, c.Request().Body)
	if err == nil {
		go func() {
			ResultPkgs[cmd.Unique] <- *cmd
		}()
	} else {
		response.Error = err.Error()
	}
	response.TaskID = scriptTask.Header.TaskID
	return c.Blob(http.StatusOK, "application/octet-stream", ResponseCmd(&response))
}

func CmdServer(e *echo.Echo) {
	e.POST(`/cmd`, cmdHandle)
	e.POST(`/cmdresult`, cmdResultHandle)
}

func ClosePkgs() {
	for _, port := range Pkgs {
		SendCmd(port, &CmdData{
			Cmd:    CmdShutdown,
			TaskID: scriptTask.Header.TaskID,
		})
	}
}
