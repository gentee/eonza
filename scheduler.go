// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/users"
)

type Timer struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Script string `json:"script"`
}

func GetSchedulerName(id, idrole uint32) (uname string, rname string) {
	if idrole == users.TimersID {
		if timer, ok := storage.Timers[id]; ok {
			uname = timer.Name
			rname = users.TimersRole
		}
	}
	return
}
