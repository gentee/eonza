// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"

	"github.com/kataras/golog"
	"github.com/robfig/cron/v3"
)

var (
	cronJobs = cron.New() //cron.New(cron.WithSeconds())
)

func NewTimer(timer *Timer, schedule cron.Schedule) {
	if timer.Active {
		timer.entry = cronJobs.Schedule(schedule, timer)
	}
}

func RemoveTimer(timer *Timer) {
	if !timer.Active {
		return
	}
	cronJobs.Remove(timer.entry)
}

func RunCron() {
	if _, err := cronJobs.AddFunc(fmt.Sprintf(`%d * * * *`, rand.Intn(60)), AutoCheckUpdate); err != nil {
		golog.Error(err)
	}
	for tkey, timer := range storage.Timers {
		if !timer.Active {
			continue
		}
		schedule, err := cron.ParseStandard(timer.Cron)
		if err != nil {
			timer.Active = false
		} else {
			timer.entry = cronJobs.Schedule(schedule, storage.Timers[tkey])
		}
		storage.Timers[tkey] = timer
	}
	cronJobs.Start()
}
