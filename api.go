// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eonza/lib"
	es "eonza/script"
	"eonza/users"

	"github.com/gentee/gentee"
	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

type CompileResponse struct {
	Success bool   `json:"success"`
	Source  string `json:"source,omitempty"`
	Error   string `json:"error,omitempty"`
}

type RunResponse struct {
	Success bool   `json:"success"`
	Port    int    `json:"port"`
	ID      uint32 `json:"id"`
	Error   string `json:"error,omitempty"`
}

type TaskStatus struct {
	TaskID  uint32 `json:"taskid"`
	Status  int    `json:"status"`
	Message string `json:"msg,omitempty"`
	Time    int64  `json:"time,omitempty"`
}

type TaskInfo struct {
	ID         uint32 `json:"id"`
	Status     int    `json:"status"`
	Name       string `json:"name"`
	StartTime  string `json:"start"`
	FinishTime string `json:"finish"`
	//	UserID     uint32 `json:"userid"`
	//	RoleID     uint32 `json:"roleid"`
	User    string `json:"user"`
	Role    string `json:"role"`
	ToDel   bool   `json:"todel"`
	Port    int    `json:"port"`
	Message string `json:"message,omitempty"`
}

type TasksResponse struct {
	List     []TaskInfo `json:"list,omitempty"`
	Page     int        `json:"page"`
	AllPages int        `json:"allpages"`
	Error    string     `json:"error,omitempty"`
}

type Feedback struct {
	Like     int    `json:"like"`
	Feedback string `json:"feedback"`
	Email    string `json:"email"`
}

func jsonError(c echo.Context, err interface{}) error {
	return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
}

func jsonSuccess(c echo.Context) error {
	return c.JSON(http.StatusOK, Response{Success: true})
}

func compileHandle(c echo.Context) error {
	var (
		item *Script
		src  string
		err  error
	)
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	name := c.QueryParam(`name`)
	if item = getScript(name); item == nil {
		return jsonError(c, Lang(DefLang, `erropen`, name))
	}
	langCode := GetLangCode(c.(*Auth).User)
	title := item.Settings.Title
	if langTitle := strings.Trim(title, `#`); langTitle != title {
		if val, ok := item.Langs[langCode][langTitle]; ok {
			title = val
		} else if val, ok := item.Langs[LangDefCode][langTitle]; ok {
			title = val
		}
	}
	header := es.Header{
		Name: name,
		Lang: langCode,
	}
	if src, err = GenSource(item, &header); err != nil {
		return jsonError(c, err)
	}
	workspace := gentee.New()
	_, _, err = workspace.Compile(src, header.Name)
	src, _ = lib.Markdown("```go\r\n" + src + "\r\n```")
	if err != nil {
		return c.JSON(http.StatusOK, CompileResponse{Error: err.Error(), Source: src})
	}
	return c.JSON(http.StatusOK, CompileResponse{Success: true, Source: src})
}

func runHandle(c echo.Context) error {
	var console bool
	open := true
	if len(c.QueryParam(`silent`)) > 0 || cfg.HTTP.Host != Localhost {
		open = false
	}
	if len(c.QueryParam(`console`)) > 0 {
		console = true
	}
	user := c.(*Auth).User
	role, _ := GetRole(user.RoleID)
	rs := RunScript{
		Name:    c.QueryParam(`name`),
		Open:    open,
		Console: console,
		User:    *user,
		Role:    role,
		IP:      c.RealIP(),
	}
	if err := systemRun(&rs); err != nil {
		return jsonError(c, err)
	}
	if err := AddHistoryRun(user.ID, rs.Name); err != nil {
		return jsonError(c, err)
	}
	if console {
		return c.Blob(http.StatusOK, ``, rs.Encoded)
	}
	return c.JSON(http.StatusOK, RunResponse{Success: true, Port: rs.Port, ID: rs.ID})
}

func pingHandle(c echo.Context) error {
	return c.HTML(http.StatusOK, Success)
}

func sysTaskHandle(c echo.Context) error {
	var (
		err    error
		taskid uint64
	)
	cmd := c.QueryParam(`cmd`)
	if taskid, err = strconv.ParseUint(c.QueryParam(`taskid`), 10, 32); err != nil {
		return jsonError(c, err)
	}
	user := c.(*Auth).User
	for _, item := range tasks {
		if item.ID == uint32(taskid) {
			if user.RoleID != users.XAdminID && user.ID != item.UserID {
				return jsonError(c, fmt.Errorf(`Access denied`))
			}
			go func() {
				lib.LocalGet(item.LocalPort, fmt.Sprintf("sys?cmd=%s&taskid=%d", cmd, taskid))
			}()
			break
		}
	}
	return jsonSuccess(c)
}

func tasksHandle(c echo.Context) error {
	if err := CheckTasks(); err != nil {
		return jsonError(c, err)
	}
	list := ListTasks()
	page := 1
	allpages := 1
	listInfo := make([]TaskInfo, 0, len(list))
	user := c.(*Auth).User
	var (
		taskFlag int
	)
	if user.RoleID != users.XAdminID {
		if role, ok := GetRole(user.RoleID); ok {
			taskFlag = role.Tasks
		}
	}
	for _, item := range list {
		var finish string
		if item.FinishTime > 0 {
			finish = time.Unix(item.FinishTime, 0).Format(TimeFormat)
		} else if user.RoleID != users.XAdminID && user.ID != item.UserID {
			continue
		}
		if user.RoleID == users.XAdminID || (taskFlag&4 == 4) ||
			(taskFlag&1 == 1 && user.ID == item.UserID) ||
			(taskFlag&2 == 2 && user.RoleID == item.RoleID) {
			todel := user.RoleID == users.XAdminID || (taskFlag&0x400 == 0x400) ||
				(taskFlag&0x100 == 0x100 && user.ID == item.UserID) ||
				(taskFlag&0x200 == 0x200 && user.RoleID == item.RoleID)
			var userName, roleName string
			userName, roleName = GetUserRole(item.UserID, item.RoleID)
			listInfo = append(listInfo, TaskInfo{
				ID:         item.ID,
				Status:     item.Status,
				Name:       item.Name,
				StartTime:  time.Unix(item.StartTime, 0).Format(TimeFormat),
				FinishTime: finish,
				User:       userName,
				Role:       roleName,
				Port:       item.Port,
				ToDel:      todel,
				Message:    item.Message,
			})
		}
	}
	if len(listInfo) > 0 {
		allpages = len(listInfo) / TasksPage
		if len(listInfo)%TasksPage != 0 {
			allpages++
		}
		if cur := c.QueryParam("page"); len(cur) > 0 {
			if ipage, err := strconv.ParseInt(cur, 10, 32); err == nil && int(ipage) <= allpages {
				page = int(ipage)
			}
		}
	}
	start := TasksPage * (page - 1)
	end := TasksPage * page
	if end > len(listInfo) {
		end = len(listInfo)
	}
	return c.JSON(http.StatusOK, &TasksResponse{
		List:     listInfo[start:end],
		Page:     page,
		AllPages: allpages,
	})
}

func removeTaskHandle(c echo.Context) error {
	var (
		ptask *Task
		ok    bool
	)
	idTask, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if ptask, ok = tasks[uint32(idTask)]; !ok {
		return jsonError(c, fmt.Errorf(`task %d has not been found`, idTask))
	}
	user := c.(*Auth).User
	if user.RoleID != users.XAdminID {
		var access bool
		if role, ok := GetRole(user.RoleID); ok {
			taskFlag := role.Tasks
			access = (taskFlag&0x400 == 0x400) ||
				(taskFlag&0x100 == 0x100 && user.ID == ptask.UserID) ||
				(taskFlag&0x200 == 0x200 && user.RoleID == ptask.RoleID)
		}
		if !access {
			return jsonError(c, fmt.Errorf(`Access denied`))
		}
	}
	RemoveTask(uint32(idTask))
	if errSave := SaveTasks(); errSave != nil {
		golog.Error(errSave)
	}
	return tasksHandle(c)
}

func trialHandle(c echo.Context) error {
	var (
		err  error
		mode int
	)
	if c.(*Auth).User.RoleID != users.XAdminID {
		return jsonError(c, fmt.Errorf(`Access denied`))
	}
	mode = storage.Trial.Mode
	if c.Param("id") == `1` {
		if storage.Trial.Mode == TrialOff && storage.Trial.Count < TrialDays {
			storage.Trial.Mode = TrialOn
		}
	} else {
		if storage.Trial.Mode == TrialOn {
			storage.Trial.Mode = TrialOff
		}
	}
	if mode != storage.Trial.Mode {
		SetActive()
		if err = SaveStorage(); err != nil {
			return jsonError(c, err)
		}
	}
	return proSettingsHandle(c)
}

func feedbackHandle(c echo.Context) error {
	var (
		feedback Feedback
		resp     *http.Response
		body     []byte
		err      error
	)
	if err = c.Bind(&feedback); err != nil {
		return jsonError(c, err)
	}
	//	user := c.(*Auth).User
	jsonValue, err := json.Marshal(feedback)
	if err == nil {
		resp, err = http.Post(appInfo.Homepage+"feedback",
			"application/json", bytes.NewBuffer(jsonValue))
		if err == nil {
			if body, err = io.ReadAll(resp.Body); err == nil {
				var answer Response
				if err = json.Unmarshal(body, &answer); err == nil {
					if len(answer.Error) > 0 {
						err = fmt.Errorf(answer.Error)
					}
				}
			}
			resp.Body.Close()
		}
	}
	if err != nil {
		return jsonError(c, err)
	}
	return jsonSuccess(c)
}
