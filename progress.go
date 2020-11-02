// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"time"

	"github.com/gentee/gentee"
	"github.com/gentee/gentee/vm"
)

type ProgressData struct {
	Percent int64
	Remain  time.Duration
	Start   time.Time
	Updated time.Time
}

type ProgressInfo struct {
	ID      uint32 `json:"id"`
	Type    int32  `json:"type"`
	Total   string `json:"total"`
	Current string `json:"current"`
	Source  string `json:"source"`
	Dest    string `json:"dest"`
	Percent int64  `json:"percent"`
	Remain  string `json:"remain"`
}

func ProgressHandle(prog *gentee.Progress) bool {
	if scriptTask.Header.Console {
		return true
	}
	switch prog.Status {
	case 0:
		now := time.Now()
		prog.Custom = ProgressData{
			Start:   now,
			Updated: now,
		}
	case 1:
		custom := prog.Custom.(ProgressData)
		percent := int64(100.0 * prog.Ratio)
		if /*percent != progress.Percent &&*/ time.Since(custom.Updated) > 500*time.Millisecond {
			/*				dif := progress.Current - progress.prevTotal
							speed := float64(dif) / float64(since)
							progress.Remain = time.Duration(float64(progress.Total-progress.Current) / speed).Round(time.Second)*/
			remain := time.Duration(float64(time.Since(custom.Start)) * (1 - prog.Ratio) /
				prog.Ratio).Round(time.Second)
			if percent != custom.Percent || remain != custom.Remain {
				custom.Percent = percent
				custom.Remain = remain
				custom.Updated = time.Now()
				prog.Custom = custom
				chProgress <- prog
			}
		}
	case 2:
		custom := prog.Custom.(ProgressData)
		if custom.Start != custom.Updated {
			custom.Percent = 100
			prog.Custom = custom
			chProgress <- prog
		}
	}
	return true
}

func ProgressToString(progress *gentee.Progress) (string, error) {
	var remain string
	custom := progress.Custom.(ProgressData)
	if custom.Remain <= 24*time.Hour {
		remain = custom.Remain.String()
	}

	data, err := json.Marshal(ProgressInfo{
		ID:      progress.ID,
		Type:    progress.Type,
		Total:   vm.SizeToStr(progress.Total, ``),
		Current: vm.SizeToStr(progress.Current, ``),
		Percent: custom.Percent,
		Remain:  remain,
		Source:  progress.Source,
		Dest:    progress.Dest,
	})
	return string(data), err
}
