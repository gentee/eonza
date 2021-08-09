// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"archive/zip"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"eonza/lib"
	"eonza/script"
	"eonza/users"
	"fmt"
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

	TasksPage = 50
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
	LocalPort  int    `json:"localport"`
	Message    string `json:"message,omitempty"`
	SourceCode string `json:"sourcecode,omitempty"`
	Locked     bool   `json:"locked"`
}

var (
	traceFile      *os.File
	tasks          map[uint32]*Task
	ports          [PortsPool]bool
	prevCheckTasks time.Time
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

func CheckTasks() (err error) {
	if prevCheckTasks.Add(1 * time.Hour).Before(time.Now()) {
		list := ListTasks()
		var count int
		timeout := time.Now().AddDate(0, 0, -storage.Settings.RemoveAfter)
		for _, item := range list {
			if item.Locked {
				continue
			}
			if count > storage.Settings.MaxTasks || time.Unix(item.StartTime, 0).Before(timeout) {
				RemoveTask(item.ID)
				continue
			}
			count++
		}
		if len(list) != len(tasks) {
			err = SaveTasks()
		}
		prevCheckTasks = time.Now()
	}
	return
}

func SaveTrace(task *Task) (err error) {
	if task.Status >= TaskFinished {
		freePort(task.Port)
		freePort(task.LocalPort)
	}
	_, err = traceFile.Write([]byte(fmt.Sprintf("%s\r\n", task.String())))
	return
}

func RemoveTask(id uint32) {
	delete(tasks, id)
	for _, ext := range append(TaskExt, `zip`) {
		os.Remove(filepath.Join(cfg.Log.Dir, fmt.Sprintf("%08x.%s", id, ext)))
	}
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
	return ret
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
		LocalPort: header.HTTP.LocalPort,
	}
	if header.Role.ID >= users.ResRoleID {
		task.RoleID = header.Role.ID
	}
	if err = SaveTrace(&task); err != nil {
		return
	}
	if _, ok := tasks[task.ID]; ok {
		return fmt.Errorf(`task %x exists`, task.ID)
	}
	tasks[task.ID] = &task
	return CheckTasks()
}

func (task *Task) String() string {
	var locked string
	if task.Locked {
		locked = `*`
	}
	return fmt.Sprintf("%x,%x/%x/%s,%d,%s,%d,%d,%d%s,%s", task.ID, task.UserID, task.RoleID, task.IP,
		task.Port, task.Name,
		task.StartTime, task.FinishTime, task.Status, locked, task.Message)
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
		status := vals[6]
		if strings.HasSuffix(status, `*`) {
			task.Locked = true
			status = status[:len(status)-1]
		}
		if ival, err = strconv.ParseInt(status, 10, 64); err != nil {
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
			active := false
			body, err := lib.LocalGet(item.LocalPort, `info`)
			if err == nil {
				var task Task
				if err = json.Unmarshal(body, &task); err == nil && task.ID == item.ID {
					active = true
					tasks[key].Status = task.Status
				}
			}
			if !active {
				tasks[key].Status = TaskCrashed
				tasks[key].FinishTime = time.Now().Unix()
			}
		}
	}
	err = CheckTasks()
	return
}

func CloseTaskManager() {
	traceFile.Close()
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

func GetTaskFiles(id uint32, render bool) (ret []string, replist []script.Report) {
	var (
		err error
		out []byte
	)
	fname := fmt.Sprintf(`%08x.`, id)

	ret = make([]string, len(TaskExt))
	decode := func(i int, buf *bytes.Buffer) {
		if i == TExtReport {
			dec := gob.NewDecoder(buf)
			if err = dec.Decode(&replist); err != nil {
				golog.Error(err)
			} else if render {
				for i, item := range replist {
					replist[i].Body = script.ReportToHtml(item)
				}
			}
		} else {
			ret[i] = string(buf.Bytes())
		}
	}

	for i, ext := range TaskExt {
		if i == TExtTrace {
			continue
		}
		if out, err = os.ReadFile(filepath.Join(cfg.Log.Dir, fname+ext)); err == nil {
			decode(i, bytes.NewBuffer(out))
		}
	}
	if len(ret[TExtLog]) > 0 || len(ret[TExtOut]) > 0 || len(ret[TExtSrc]) > 0 ||
		len(ret[TExtReport]) > 0 {
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
					decode(i, &buf)
				}
			}

		}
	}
	return
}
