// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"eonza/lib"
	"eonza/script"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gentee/gentee"
	"github.com/gorilla/websocket"
	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

const (
	WcClose  = iota // close connection
	WcStatus        // change status
	WcStdout        // new line in console
	WcStdbuf        // current output including carriage
	WcLogout        // log output
	WcForm          // form output
)

const (
	TExtTrace = iota
	TExtOut
	TExtLog
	TExtSrc
)

type FormResponse struct {
	FormID uint32                 `json:"formid"`
	Values map[string]interface{} `json:"values"`
}

type WsClient struct {
	StdoutCount int
	LogoutCount int
	Conn        *websocket.Conn
}

type WsCmd struct {
	TaskID  uint32 `json:"taskid"`
	Cmd     int    `json:"cmd"`
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Time    string `json:"finish,omitempty"`
}

type StdinForm struct {
	Message string `json:"message"`
}

var (
	task       Task
	prevStatus int
	upgrader   websocket.Upgrader
	wsChan     chan WsCmd
	TaskExt    = []string{"trace", "out", "log", "g"}

	stdoutBuf []string
	logoutBuf []string
	formData  []script.FormInfo
	iStdout   int
	iLogout   int

	console   *os.File
	cmdFile   *os.File
	outFile   *os.File
	logScript *os.File

	chStdin    chan []byte
	chStdout   chan []byte
	chLogout   chan string
	chForm     chan script.FormInfo
	chFormNext chan bool
	chSystem   chan int
	chFinish   chan bool

	clients = make(map[uint32]WsClient)
)

func closeTask() {
	var files []string

	for ; iStdout < len(stdoutBuf); iStdout++ {
		out := lib.ClearCarriage(stdoutBuf[iStdout]) + "\r\n"
		if _, err := outFile.Write([]byte(out)); err != nil {
			golog.Error(err)
		}
	}
	cmdFile.Close()
	outFile.Close()
	logScript.Close()
	for i, item := range TaskExt {
		if i == TExtSrc && len(scriptTask.Header.SourceCode) == 0 {
			continue
		}
		files = append(files, filepath.Join(scriptTask.Header.LogDir,
			fmt.Sprintf("%08x.%s", task.ID, item)))
	}
	output := filepath.Join(scriptTask.Header.LogDir, fmt.Sprintf("%08x.zip", task.ID))

	if err := lib.ZipFiles(output, files); err != nil {
		golog.Error(err)
	} else {
		for _, item := range files {
			os.Remove(item)
		}
	}
}

func sendForm(client WsClient) error {
	if len(formData) == 0 {
		return nil
	}
	return client.Conn.WriteJSON(WsCmd{
		TaskID:  task.ID,
		Cmd:     WcForm,
		Message: formData[0].Data,
		Status:  int(formData[0].ID),
	})
}

func sendStdout(client WsClient) error {
	for i := client.StdoutCount; i < iStdout; i++ {
		if err := client.Conn.WriteJSON(WsCmd{
			TaskID:  task.ID,
			Cmd:     WcStdout,
			Message: stdoutBuf[i],
		}); err != nil {
			return err
		}
	}
	return client.Conn.WriteJSON(WsCmd{
		TaskID:  task.ID,
		Cmd:     WcStdbuf,
		Message: lib.ClearCarriage(stdoutBuf[iStdout]),
	})
}

func sendLogout(client WsClient) error {
	for i := client.LogoutCount; i < iLogout; i++ {
		if err := client.Conn.WriteJSON(WsCmd{
			TaskID:  task.ID,
			Cmd:     WcLogout,
			Message: logoutBuf[i],
		}); err != nil {
			return err
		}
	}
	return nil
}
func initTask() script.Settings {
	var err error

	task = Task{
		ID:        scriptTask.Header.TaskID,
		UserID:    scriptTask.Header.UserID,
		Status:    TaskActive,
		Name:      scriptTask.Header.Name,
		StartTime: time.Now().Unix(),
		Port:      scriptTask.Header.HTTP.Port,
	}

	createFile := func(ext string) *os.File {
		ret, err := os.OpenFile(filepath.Join(scriptTask.Header.LogDir,
			fmt.Sprintf(`%08x.`+ext, task.ID)), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0666)
		if err != nil {
			golog.Fatal(err)
		}
		return ret
	}

	cmdFile = createFile(`trace`)
	outFile = createFile(`out`)
	logScript = createFile(`log`)

	if _, err = cmdFile.Write([]byte(task.Head())); err != nil {
		golog.Fatal(err)
	}
	if len(scriptTask.Header.SourceCode) > 0 {
		var out []byte
		if out, err = lib.GzipDecompress(scriptTask.Header.SourceCode); err != nil {
			golog.Fatal(err)
		}
		task.SourceCode = string(out)
		srcFile := createFile(`g`)
		if _, err = srcFile.Write(out); err != nil {
			golog.Fatal(err)
		}
		srcFile.Close()
	}
	console = os.Stdout
	upgrader = websocket.Upgrader{}
	wsChan = make(chan WsCmd)

	chStdin = make(chan []byte)
	chStdout = make(chan []byte)
	chLogout = make(chan string)
	chForm = make(chan script.FormInfo)
	chFormNext = make(chan bool)
	chSystem = make(chan int)
	chFinish = make(chan bool)
	stdoutBuf = []string{``}
	logoutBuf = make([]string, 0, 32)

	go func() {
		var out []byte
		for {
			out = <-chStdout
			mutex.Lock()
			off := len(stdoutBuf) - 1
			lines := strings.Split(string(out), "\n")
			stdoutBuf[off] += lines[0]
			for i := 1; i < len(lines)-1; i++ {
				lines[i] = lib.ClearCarriage(lines[i])
				stdoutBuf = append(stdoutBuf, lines[i])
			}
			if len(lines) > 1 {
				stdoutBuf[off] = lib.ClearCarriage(stdoutBuf[off])
				stdoutBuf = append(stdoutBuf, lines[len(lines)-1])
			}
			for i := off; i < len(stdoutBuf)-1; i++ {
				if _, err := outFile.Write([]byte(stdoutBuf[i] + "\r\n")); err != nil {
					golog.Error(err)
				}
			}
			iStdout = len(stdoutBuf) - 1
			for id, client := range clients {
				if sendStdout(client) == nil {
					client.StdoutCount = iStdout
					clients[id] = client
				} else {
					client.Conn.Close()
					delete(clients, id)
				}
			}
			mutex.Unlock()
		}
	}()

	go func() {
		var out string
		for {
			out = <-chLogout
			mutex.Lock()
			if _, err := logScript.Write([]byte(out + "\r\n")); err != nil {
				golog.Error(err)
			}
			logoutBuf = append(logoutBuf, out)
			iLogout = len(logoutBuf)
			for id, client := range clients {
				if sendLogout(client) == nil {
					client.LogoutCount = iLogout
					clients[id] = client
				} else {
					client.Conn.Close()
					delete(clients, id)
				}
			}
			mutex.Unlock()
		}
	}()

	go func() {
		var out script.FormInfo
		for {
			select {
			case out = <-chForm:
				formData = append(formData, out)
				if len(formData) > 1 {
					continue
				}
			case <-chFormNext:
			}
			mutex.Lock()
			for id, client := range clients {
				if sendForm(client) != nil {
					client.Conn.Close()
					delete(clients, id)
				}
			}
			mutex.Unlock()
		}
	}()

	go func() {
		var cmd WsCmd
		for task.Status <= TaskSuspended {
			cmd = <-wsChan
			mutex.Lock()
			for id, client := range clients {
				err := client.Conn.WriteJSON(cmd)
				if err != nil {
					client.Conn.Close()
					delete(clients, id)
				}
			}
			mutex.Unlock()
		}
		for _, client := range clients {
			client.Conn.Close()
		}
		chFinish <- true
	}()
	var langid int
	for i, lang := range langs {
		if lang == scriptTask.Header.Lang {
			langid = i
			break
		}
	}
	glob := &langRes[langid]
	for name, val := range scriptTask.Header.Constants {
		(*glob)[name] = val
	}
	if exfile, err := os.Executable(); err != nil {
		golog.Fatal(err)
	} else {
		(*glob)[`apppath`] = filepath.Dir(exfile)
	}
	(*glob)[`temppath`] = os.TempDir()
	(*glob)[`os`] = runtime.GOOS
	(*glob)[`isconsole`] = fmt.Sprint(scriptTask.Header.Console)
	(*glob)[`port`] = fmt.Sprint(scriptTask.Header.HTTP.Port)

	script.InitData(chLogout, chForm, glob)
	return script.Settings{
		ChStdin:  chStdin,
		ChStdout: chStdout,
		ChSystem: chSystem,
	}
}

func wsTaskHandle(c echo.Context) error {
	//	var cmd WsCmd
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	if err = ws.WriteJSON(WsCmd{
		TaskID: task.ID,
		Cmd:    WcStatus,
		Status: task.Status,
	}); err == nil {
		client := WsClient{
			Conn: ws,
		}
		if err = sendStdout(client); err == nil {
			client.StdoutCount = iStdout
			if err = sendLogout(client); err == nil {
				client.LogoutCount = iLogout
				if err = sendForm(client); err == nil {
					clients[lib.RndNum()] = client
				}
			}
		}
		if err != nil {
			client.Conn.Close()
		}
	}
	return nil
}

func infoHandle(c echo.Context) error {
	return c.JSON(http.StatusOK, task)
}

func sendCmdStatus(status int, timeStamp int64, message string) {
	taskTrace(timeStamp, status, message)
	jsonValue, err := json.Marshal(TaskStatus{
		TaskID:  task.ID,
		Status:  status,
		Message: message,
		Time:    timeStamp,
	})
	if err == nil {
		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/api/taskstatus",
			scriptTask.Header.ServerPort), "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			golog.Error(err)
		} else {
			resp.Body.Close()
		}
	}
	var finish string
	task.Status = status
	if task.Status >= TaskFinished {
		task.FinishTime = timeStamp
		finish = time.Unix(timeStamp, 0).Format(TimeFormat)
	}
	wsChan <- WsCmd{TaskID: task.ID, Cmd: WcStatus, Status: status, Message: message, Time: finish}
}

func sysHandle(c echo.Context) error {
	cmd, _ := strconv.ParseInt(c.QueryParam(`cmd`), 10, 64)
	id, _ := strconv.ParseInt(c.QueryParam(`taskid`), 10, 64)
	if uint32(id) != task.ID {
		return jsonError(c, fmt.Errorf(`wrong task id`))
	}
	if cmd == gentee.SysTerminate {
		go func() {
			setStatus(TaskTerminated)
			closeTask()
			<-chFinish
			os.Exit(1)
		}()
	}
	if cmd >= gentee.SysSuspend && cmd < gentee.SysTerminate {
		chSystem <- int(cmd)
		switch cmd {
		case gentee.SysSuspend:
			prevStatus = task.Status
			sendCmdStatus(TaskSuspended, time.Now().Unix(), ``)
		case gentee.SysResume:
			sendCmdStatus(prevStatus, time.Now().Unix(), ``)
		}
	}
	return jsonSuccess(c)
}

func setStatus(status int, pars ...interface{}) {
	var message string
	//	cmd := WsCmd{TaskID: task.ID, Cmd: WcStatus, Status: status}
	if len(pars) > 0 {
		message = fmt.Sprint(pars...)
		task.Message = message
	}
	sendCmdStatus(status, time.Now().Unix(), message)

	/*	taskTrace(task.FinishTime, status, task.Message)

		jsonValue, err := json.Marshal(TaskStatus{
			TaskID:  task.ID,
			Status:  task.Status,
			Message: task.Message,
			Time:    task.FinishTime,
		})
		if err == nil {
			resp, err := http.Post(fmt.Sprintf("http://localhost:%d/api/taskstatus",
				scriptTask.Header.ServerPort), "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				golog.Error(err)
			} else {
				resp.Body.Close()
			}
		}

	 	wsChan <- cmd */
}

func debug(pars ...interface{}) {
	console.Write([]byte(fmt.Sprintln(pars...)))
}

func stdinHandle(c echo.Context) error {
	var (
		form StdinForm
		err  error
	)
	id, _ := strconv.ParseInt(c.QueryParam(`taskid`), 10, 64)
	if uint32(id) != task.ID {
		return jsonError(c, fmt.Errorf(`wrong task id`))
	}
	if err = c.Bind(&form); err != nil {
		return jsonError(c, err)
	}
	msg := form.Message + "\n"
	chStdin <- []byte(msg)
	chStdout <- []byte(msg)
	return jsonSuccess(c)
}

func formHandle(c echo.Context) error {
	var (
		form FormResponse
		err  error
	)
	id, _ := strconv.ParseInt(c.QueryParam(`taskid`), 10, 64)
	if uint32(id) != task.ID {
		return jsonError(c, fmt.Errorf(`wrong task id`))
	}
	if err = c.Bind(&form); err != nil {
		return jsonError(c, err)
	}
	if len(formData) > 0 && formData[0].ID == form.FormID {
		for key, val := range form.Values {
			script.SetVar(key, fmt.Sprint(val))
		}
		if forLog, err := json.Marshal(form.Values); err != nil {
			script.LogOutput(script.LOG_ERROR, err.Error())
		} else {
			script.LogOutput(script.LOG_FORM, string(forLog))
		}
		formData[0].ChResponse <- true
		formData = formData[1:]
		if len(formData) > 0 {
			chFormNext <- true
		}
	}
	return jsonSuccess(c)
}
