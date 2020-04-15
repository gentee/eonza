// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"

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
	tasks []*Task
	ports [PortsPool]bool
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
