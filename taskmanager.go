// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/script"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/golog"
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
	ID         uint32
	Status     int
	Name       string
	StartTime  int64
	FinishTime int64
	UserID     uint32
	Port       int
	Message    string
}

var (
	traceFile *os.File
	tasks     map[uint32]*Task
	ports     [PortsPool]bool
)

func (task *Task) Head() string {
	return fmt.Sprintf("%x,%x,%d,%s,%d\r\n", task.ID, task.UserID, task.Port, task.Name, task.StartTime)
}

func taskTrace(unixTime int64, status int, message string) {
	out := fmt.Sprintf("%d,%x,%s\r\n", unixTime, status, message)

	if _, err := cmdFile.Write([]byte(out)); err != nil {
		golog.Fatal(err)
	}
}

func SaveTrace(task *Task) (err error) {
	_, err = traceFile.Write([]byte(fmt.Sprintf("%s\r\n", task.String())))
	return
}

func NewTask(header script.Header) (err error) {
	task := Task{
		ID:        header.TaskID,
		Status:    TaskActive,
		Name:      header.Name,
		StartTime: time.Now().Unix(),
		UserID:    header.UserID,
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
	return fmt.Sprintf("%x,%x,%d,%s,%d,%d,%d,%s", task.ID, task.UserID, task.Port, task.Name,
		task.StartTime, task.FinishTime, task.Status, task.Message)
}

func LogToTask(input string) (task Task, err error) {
	var (
		uival uint64
		ival  int64
	)
	vals := strings.Split(strings.TrimSpace(input), `,`)
	if len(vals) == 8 {
		if uival, err = strconv.ParseUint(vals[0], 16, 32); err != nil {
			return
		}
		task.ID = uint32(uival)
		if uival, err = strconv.ParseUint(vals[1], 16, 32); err != nil {
			return
		}
		task.UserID = uint32(uival)
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
	}
	return
}

func InitTaskManager() (err error) {
	traceFile, err = os.OpenFile(filepath.Join(cfg.Log.Dir,
		fmt.Sprintf(`tasks.trace`)), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	tasks = make(map[uint32]*Task)
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
