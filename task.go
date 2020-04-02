// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import "fmt"

func sendStatus(status int, pars ...interface{}) {
	task := TaskStatus{
		TaskID: scriptTask.Header.TaskID,
		Status: status,
	}
	if len(pars) > 0 {
		task.Message = fmt.Sprint(pars[0])
	}
}
