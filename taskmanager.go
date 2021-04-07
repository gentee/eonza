// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"eonza/lib"
	"eonza/script"
	"eonza/users"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

const ( // TaskStatus
	TaskStart = iota
	TaskActive
	TaskWaiting // waiting for the user's action
	TaskSuspended
	TaskFinished
	TaskTerminated
	TaskFailed
	TaskCrashed
)

type Task struct {
	ID         uint32 `json:"id"`
	Status     int    `json:"status"`
	Name       string `json:"name"`
	IP         string `json:"ip"`
	StartTime  int64  `json:"start"`
	FinishTime int64  `json:"finish"`
	UserID     uint32 `json:"userid"`
	RoleID     uint32 `json:"roleid"`
	Port       int    `json:"port"`
	Message    string `json:"message,omitempty"`
	SourceCode string `json:"sourcecode,omitempty"`
}

var (
	traceFile *os.File
	tasks     map[uint32]*Task
	ports     [PortsPool]bool
)

func (task *Task) Head() string {
	return fmt.Sprintf("%x,%x/%x/%s,%d,%s,%d\r\n", task.ID, task.UserID, task.RoleID, task.IP,
		task.Port, task.Name, task.StartTime)
}

func taskTrace(unixTime int64, status int, message string) {
	out := fmt.Sprintf("%d,%x,%s\r\n", unixTime, status, message)

	if _, err := cmdFile.Write([]byte(out)); err != nil {
		golog.Fatal(err)
	}
}

func SaveTrace(task *Task) (err error) {
	if task.Status >= TaskFinished {
		freePort(task.Port)
	}
	_, err = traceFile.Write([]byte(fmt.Sprintf("%s\r\n", task.String())))
	if len(tasks) > int(TasksLimit*1.2) {
		if errSave := SaveTasks(); errSave != nil {
			golog.Error(errSave)
		}
	}
	return
}

func RemoveTask(id uint32) {
	for _, ext := range append(TaskExt, `zip`) {
		os.Remove(filepath.Join(cfg.Log.Dir, fmt.Sprintf("%08x.%s", id, ext)))
	}
	/*	if err := filepath.Walk(cfg.Log.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasPrefix(info.Name(), pref) {
				os.Remove(path)
			}
			return nil
		}); err != nil {
			golog.Error(err)
		}*/
}

func GetTaskName(id uint32) (ret string) {
	if v, ok := tasks[id]; ok {
		ret = v.Name
	} else {
		ret = strconv.FormatUint(uint64(id), 16)
	}
	return
}

func ListTasks() []*Task {
	ret := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		ret = append(ret, task)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].StartTime == ret[j].StartTime {
			return ret[i].FinishTime > ret[j].FinishTime
		}
		return ret[i].StartTime > ret[j].StartTime
	})
	if len(ret) > TasksLimit {
		for i := TasksLimit; i < len(ret); i++ {
			delete(tasks, ret[i].ID)
			RemoveTask(ret[i].ID)
		}
		ret = ret[:TasksLimit]
	}
	return ret
}

func SaveTasks() (err error) {
	list := ListTasks()
	var out string
	for i := len(list) - 1; i >= 0; i-- {
		out += list[i].String() + "\r\n"
	}
	traceFile.Truncate(0)
	traceFile.Seek(0, 0)
	_, err = traceFile.Write([]byte(out))
	return
}

func NewTask(header script.Header) (err error) {
	task := Task{
		ID:        header.TaskID,
		Status:    TaskActive,
		Name:      header.Name,
		IP:        header.IP,
		StartTime: time.Now().Unix(),
		UserID:    header.User.ID,
		RoleID:    header.User.RoleID,
		Port:      header.HTTP.Port,
	}
	if err = SaveTrace(&task); err != nil {
		return
	}
	if _, ok := tasks[task.ID]; ok {
		return fmt.Errorf(`task %x exists`, task.ID)
	}
	tasks[task.ID] = &task
	return
}

func (task *Task) String() string {
	return fmt.Sprintf("%x,%x/%x/%s,%d,%s,%d,%d,%d,%s", task.ID, task.UserID, task.RoleID, task.IP,
		task.Port, task.Name,
		task.StartTime, task.FinishTime, task.Status, task.Message)
}

func LogToTask(input string) (task Task, err error) {
	var (
		uival uint64
		ival  int64
	)
	vals := strings.Split(strings.TrimSpace(input), `,`)
	if len(vals) == 8 && len(vals[3]) > 0 {
		if uival, err = strconv.ParseUint(vals[0], 16, 32); err != nil {
			return
		}
		task.ID = uint32(uival)
		ur := strings.Split(vals[1], `/`)
		if uival, err = strconv.ParseUint(ur[0], 16, 32); err != nil {
			return
		}
		task.UserID = uint32(uival)
		if len(ur) > 1 {
			if uival, err = strconv.ParseUint(ur[1], 16, 32); err != nil {
				return
			}
			task.RoleID = uint32(uival)
			if len(ur) > 2 {
				task.IP = ur[2]
			}
		} else {
			task.RoleID = users.XAdminID
		}
		if uival, err = strconv.ParseUint(vals[2], 10, 32); err != nil {
			return
		}
		task.Port = int(uival)
		task.Name = vals[3]
		if ival, err = strconv.ParseInt(vals[4], 10, 64); err != nil {
			return
		}
		task.StartTime = ival
		if ival, err = strconv.ParseInt(vals[5], 10, 64); err != nil {
			return
		}
		task.FinishTime = ival
		if ival, err = strconv.ParseInt(vals[6], 10, 64); err != nil {
			return
		}
		task.Status = int(ival)
		task.Message = vals[7]
	} else {
		err = fmt.Errorf(`wrong task trace %s`, input)
	}
	return
}

func InitTaskManager() (err error) {
	filename := filepath.Join(cfg.Log.Dir, `tasks.trace`)
	traceFile, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	tasks = make(map[uint32]*Task)
	input, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	for _, item := range strings.Split(string(input), "\n") {
		if task, err := LogToTask(strings.TrimSpace(item)); err == nil {
			tasks[task.ID] = &task
		}
	}
	for key, item := range tasks {
		if item.Status < TaskFinished {
			url := fmt.Sprintf("http://%s:%d", Localhost, item.Port)
			resp, err := http.Get(url + `/info`)
			active := false
			if err == nil {
				if body, err := io.ReadAll(resp.Body); err == nil {
					var task Task
					if err = json.Unmarshal(body, &task); err == nil && task.ID == item.ID {
						active = true
						tasks[key].Status = task.Status
					}
					resp.Body.Close()
				}
			}
			if !active {
				tasks[key].Status = TaskCrashed
				tasks[key].FinishTime = time.Now().Unix()
			}
		}
	}
	err = SaveTasks()
	return
}

func CloseTaskManager() {
	traceFile.Close()
}

func usePort(port int) {
	i := port - cfg.HTTP.Port - 1
	if i < PortsPool {
		ports[i] = true
	}
}

func freePort(port int) {
	i := port - cfg.HTTP.Port - 1
	if i < PortsPool {
		ports[i] = false
	}
}

func getPort() (int, error) {
	var (
		i    int
		port int
	)
	for ; i < PortsPool; i++ {
		if !ports[i] {
			port = cfg.HTTP.Port + 1 + i
			if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
				_ = ln.Close()
				ports[i] = true
				break
			}
		}
	}
	if i == PortsPool {
		return i, fmt.Errorf(`There is not available port in the pool`)
	}
	return port, nil
}

func wsMainHandle(c echo.Context) error {

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	user := c.(*Auth).User
	clients[lib.RndNum()] = WsClient{
		Conn:   ws,
		UserID: user.ID,
		RoleID: user.RoleID,
	}
	return nil
}

func GetTaskFiles(id uint32) (ret []string) {
	var (
		err error
		out []byte
	)
	fname := fmt.Sprintf(`%08x.`, id)

	ret = make([]string, TExtSrc+1)
	for i, ext := range TaskExt {
		if i == TExtTrace {
			continue
		}
		if out, err = os.ReadFile(filepath.Join(cfg.Log.Dir, fname+ext)); err == nil {
			ret[i] = string(out)
		}
	}
	if len(ret[TExtLog]) > 0 && len(ret[TExtOut]) > 0 && len(ret[TExtSrc]) > 0 {
		return
	}
	r, err := zip.OpenReader(filepath.Join(cfg.Log.Dir, fname+`zip`))
	if err != nil {
		return
	}
	defer func() {
		r.Close()
	}()
	for _, f := range r.File {
		for i, ext := range TaskExt {
			if i == TExtTrace {
				continue
			}
			if len(ret[i]) == 0 && f.Name == fname+ext {
				rc, err := f.Open()
				if err != nil {
					break
				}
				var buf bytes.Buffer
				_, err = buf.ReadFrom(rc)
				rc.Close()
				if err == nil {
					ret[i] = string(buf.Bytes())
				}
			}

		}
	}
	return
}
