// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"github.com/kataras/golog"
	"github.com/robfig/cron/v3"
)

var (
	cronJobs = cron.New() //cron.New(cron.WithSeconds())
)

func RunCron() {
	if _, err := cronJobs.AddFunc(`0 * * * *`, AutoCheckUpdate); err != nil {
		golog.Error(err)
	}
	cronJobs.Start()
}
