// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"time"
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
	StartTime  time.Time
	FinishTime time.Time
	UserID     uint32
	Port       int
	Message    string
}

var (
	tasks []*Task
	ports [PortsPool]bool
)

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
