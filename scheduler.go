// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"eonza/users"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
)

type TimerCommon struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Script string `json:"script"`
	Cron   string `json:"cron"`
	Active bool   `json:"active"`
}

type TimerInfo struct {
	TimerCommon
	NextRun string `json:"next"`

	next time.Time
}

type Timer struct {
	TimerCommon

	entry cron.EntryID
}

type TimersResponse struct {
	List  []TimerInfo `json:"list"`
	Error string      `json:"error,omitempty"`
}

type Event struct {
	ID        uint32 `json:"id"`
	Name      string `json:"name"`
	Script    string `json:"script"`
	Token     string `json:"token"`
	Whitelist string `json:"whitelist"`
	Active    bool   `json:"active"`
}

type EventsResponse struct {
	List  []*Event `json:"list"`
	Error string   `json:"error,omitempty"`
}

func GetSchedulerName(id, idrole uint32) (uname string, rname string) {
	switch idrole {
	case users.TimersID:
		if timer, ok := storage.Timers[id]; ok {
			uname = timer.Name
		}
		rname = users.TimersRole
	case users.ScriptsID:
		if v, ok := tasks[id]; ok {
			uname = v.Name
		}
		rname = users.ScriptsRole
	}

	return
}

func (timer *Timer) Run() {
	if cfg.playground {
		NewNotification(&Notification{
			Text:   `Scheduler can't run scripts in playground mode`,
			UserID: timer.ID,
			RoleID: users.TimersID,
			Script: timer.Script,
		})
		return
	}
	rs := RunScript{
		Name: timer.Script,
		User: users.User{
			ID:       timer.ID,
			Nickname: timer.Name,
			RoleID:   users.TimersID,
		},
		Role: users.Role{
			ID:   users.TimersID,
			Name: users.TimersRole,
		},
		IP: Localhost,
	}
	if err := systemRun(&rs); err != nil {
		NewNotification(&Notification{
			Text:   fmt.Sprintf(`Scheduler error: %s`, err.Error()),
			UserID: timer.ID,
			RoleID: users.TimersID,
			Script: rs.Name,
		})
	}
}

func timersResponse(c echo.Context) error {
	listInfo := make([]TimerInfo, 0, len(storage.Timers))
	for _, item := range storage.Timers {
		var timer TimerInfo

		timer.TimerCommon = item.TimerCommon
		if item.Active {
			timer.next = cronJobs.Entry(item.entry).Next
			timer.NextRun = timer.next.Format(TimeFormat)
		}
		listInfo = append(listInfo, timer)
	}
	sort.Slice(listInfo, func(i, j int) bool {
		if !listInfo[i].Active {
			if listInfo[j].Active {
				return false
			}
			return strings.ToLower(listInfo[i].Name) < strings.ToLower(listInfo[j].Name)
		}
		if !listInfo[j].Active {
			return true
		}
		return listInfo[i].next.Before(listInfo[j].next)
	})
	return c.JSON(http.StatusOK, &TimersResponse{
		List: listInfo,
	})
}

func timersHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	return timersResponse(c)
}

func saveTimerHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	var timer TimerInfo
	if err := c.Bind(&timer); err != nil {
		return jsonError(c, err)
	}
	if len(timer.Script) == 0 {
		return jsonError(c, Lang(DefLang, `errreq`, `Script`))
	}
	for _, item := range storage.Timers {
		if len(timer.Name) > 0 && strings.ToLower(timer.Name) == strings.ToLower(item.Name) &&
			timer.ID != item.ID {
			return jsonError(c, fmt.Errorf(`Timer '%s' exists`, timer.Name))
		}
	}
	var (
		schedule cron.Schedule
		err      error
	)
	if schedule, err = cron.ParseStandard(timer.Cron); err != nil {
		return jsonError(c, err)
	}
	if timer.ID == 0 {
		for {
			timer.ID = lib.RndNum()
			if _, ok := storage.Timers[timer.ID]; !ok {
				break
			}
		}
	} else if curtimer, ok := storage.Timers[timer.ID]; !ok {
		return jsonError(c, fmt.Errorf(`Access denied`))
	} else {
		RemoveTimer(curtimer)
	}
	var itimer Timer
	itimer.TimerCommon = timer.TimerCommon
	NewTimer(&itimer, schedule)
	storage.Timers[itimer.ID] = &itimer
	if err := SaveStorage(); err != nil {
		return jsonError(c, err)
	}
	return timersResponse(c)
}

func removeTimerHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}

	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if timer, ok := storage.Timers[uint32(id)]; ok {
		RemoveTimer(timer)
		delete(storage.Timers, uint32(id))
		if err := SaveStorage(); err != nil {
			return jsonError(c, err)
		}
	}
	return timersResponse(c)
}

func eventsResponse(c echo.Context) error {
	listInfo := make([]*Event, 0, len(storage.Events))
	for _, item := range storage.Events {
		listInfo = append(listInfo, item)
	}
	sort.Slice(listInfo, func(i, j int) bool {
		if !listInfo[i].Active {
			if listInfo[j].Active {
				return false
			}
			return strings.ToLower(listInfo[i].Name) < strings.ToLower(listInfo[j].Name)
		}
		if !listInfo[j].Active {
			return true
		}
		return strings.ToLower(listInfo[i].Name) < strings.ToLower(listInfo[j].Name)
	})
	return c.JSON(http.StatusOK, &EventsResponse{
		List: listInfo,
	})
}

func eventsHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	return eventsResponse(c)
}

func saveEventHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	var event Event
	if err := c.Bind(&event); err != nil {
		return jsonError(c, err)
	}
	if len(event.Script) == 0 {
		return jsonError(c, Lang(DefLang, `errreq`, `Script`))
	}
	if len(event.Name) == 0 {
		return jsonError(c, Lang(DefLang, `errreq`, `Name`))
	}
	var curKey string
	for _, item := range storage.Events {
		if strings.ToLower(event.Name) == strings.ToLower(item.Name) && event.ID != item.ID {
			return jsonError(c, fmt.Errorf(`Event '%s' exists`, event.Name))
		}
		if item.ID == event.ID {
			curKey = item.Name
		}
	}
	isEvent := func(id uint32) bool {
		for _, item := range storage.Events {
			if item.ID == id {
				return true
			}
		}
		return false
	}
	if event.ID == 0 {
		for {
			event.ID = lib.RndNum()
			if !isEvent(event.ID) {
				break
			}
		}
	} else if len(curKey) == 0 {
		return jsonError(c, fmt.Errorf(`Access denied`))
	}
	if len(curKey) > 0 && curKey != event.Name {
		delete(storage.Events, curKey)
	}
	storage.Events[event.Name] = &event
	if err := SaveStorage(); err != nil {
		return jsonError(c, err)
	}
	return eventsResponse(c)
}

func removeEventHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}

	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	for key, item := range storage.Events {
		if item.ID == uint32(id) {
			delete(storage.Events, key)
			if err := SaveStorage(); err != nil {
				return jsonError(c, err)
			}
			break
		}
	}
	return eventsResponse(c)
}
