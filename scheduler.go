// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"eonza/lib"
	"eonza/users"
	"fmt"
	"net"
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

type EventData struct {
	Name string `json:"name" form:"name"`
	Data string `json:"data" form:"data"`
	Rand string `json:"rand" form:"rand"`
	Sign string `json:"sign" form:"sign"`
}

type EventsResponse struct {
	List  []*Event `json:"list"`
	Error string   `json:"error,omitempty"`
}

type RandID struct {
	ID   uint32
	Time time.Time
}

type RandResponse struct {
	Rand  string `json:"rand"`
	Error string `json:"error,omitempty"`
}

const RandLimit = 64

var (
	randIDs [RandLimit]RandID
)

func GetSchedulerName(id, idrole uint32) (uname string, rname string) {
	switch idrole {
	case users.BrowserID:
		if user, ok := GetUser(id); ok {
			uname = user.Nickname
		}
		//		uname = users.RootUser
		if len(uname) == 0 {
			uname = fmt.Sprintf("%x", id)
		}
		rname = users.BrowserRole
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
	case users.EventsID:
		for _, event := range storage.Events {
			if event.ID == id {
				uname = event.Name
				break
			}
		}
		rname = users.EventsRole
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

func eventHandle(c echo.Context) error {
	var (
		err       error
		eventData EventData
		event     *Event
		ok        bool
	)
	if err = c.Bind(&eventData); err != nil {
		return jsonError(c, err)
	}
	if event, ok = storage.Events[eventData.Name]; !ok || !event.Active {
		return AccessDenied(http.StatusForbidden)
	}
	ip := c.RealIP()
	if len(strings.TrimSpace(event.Whitelist)) > 0 {
		whitelist := strings.Split(strings.ReplaceAll(event.Whitelist, `,`, ` `), ` `)
		var matched bool
		clientip := net.ParseIP(ip)
		for _, item := range whitelist {
			if len(item) == 0 {
				continue
			}
			if item == ip {
				matched = true
				break
			}
			_, network, err := net.ParseCIDR(item)
			if err == nil && network.Contains(clientip) {
				matched = true
				break
			}
		}
		if !matched {
			return AccessDenied(http.StatusForbidden)
		}
	}
	host := c.Request().Host
	if offPort := strings.LastIndex(c.Request().Host, `:`); offPort > 0 {
		host = host[:offPort]
	}
	if !lib.IsLocalhost(host, ip) || len(eventData.Rand) > 0 {
		if len(event.Token) == 0 {
			return AccessDenied(http.StatusForbidden)
		}
		var isRnd bool
		now := time.Now()
		rnd, _ := strconv.ParseUint(eventData.Rand, 10, 32)
		if rnd > 0 {
			for i := 0; i < RandLimit; i++ {
				if uint64(randIDs[i].ID) == rnd {
					if randIDs[i].Time.After(now) {
						randIDs[i].ID = 0
						isRnd = true
						break
					}
				}
			}
		}
		if !isRnd {
			return AccessDenied(http.StatusForbidden)
		}
		shaHash := sha256.Sum256([]byte(event.Name + eventData.Data + eventData.Rand + event.Token))
		if strings.ToLower(eventData.Sign) != strings.ToLower(hex.EncodeToString(shaHash[:])) {
			return AccessDenied(http.StatusForbidden)
		}
	}
	rs := RunScript{
		Name:    event.Script,
		Open:    false,
		Console: false,
		Data:    eventData.Data,
		User: users.User{
			ID:       event.ID,
			Nickname: event.Name,
			RoleID:   users.EventsID,
		},
		Role: users.Role{
			ID:   users.EventsID,
			Name: users.EventsRole,
		},
		IP: ip,
	}
	if err := systemRun(&rs); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, RunResponse{Success: true, Port: rs.Port, ID: rs.ID})
}

func randidHandle(c echo.Context) error {
	var (
		rnd, i uint32
		rand   RandResponse
	)

	for rnd == 0 {
		rnd = lib.RndNum()
	}
	now := time.Now()
	for i = 0; i < RandLimit; i++ {
		if randIDs[i].ID == 0 || randIDs[i].Time.Before(now) {
			randIDs[i].ID = rnd
			randIDs[i].Time = now.Add(3 * time.Second)
			break
		}
	}
	if i >= RandLimit {
		rand.Error = `Too many randid requests`
	} else {
		rand.Rand = strconv.FormatUint(uint64(rnd), 10)
	}
	return c.JSON(http.StatusOK, rand)
}
