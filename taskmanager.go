// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import "time"

const ( // TaskStatus
	TaskActive  = iota
	TaskWaiting // waiting for the user's action
	TaskPaused
	TaskFinished
)

type Task struct {
	Status     int
	Name       string
	StartTime  time.Time
	FinishTime time.Time
}

var (
	tasks []*Task
)
