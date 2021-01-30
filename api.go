// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eonza/lib"
	"eonza/script"

	"github.com/gentee/gentee"
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
	UserID     uint32 `json:"userid"`
	Port       int    `json:"port"`
	Message    string `json:"message,omitempty"`
}

type TasksResponse struct {
	List  []TaskInfo `json:"list,omitempty"`
	Error string     `json:"error,omitempty"`
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
	header := script.Header{
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
	var (
		item    *Script
		src     string
		console bool
	)
	open := true
	name := c.QueryParam(`name`)
	if len(c.QueryParam(`silent`)) > 0 || cfg.HTTP.Host != Localhost {
		open = false
	}
	if len(c.QueryParam(`console`)) > 0 {
		console = true
	}
	port, err := getPort()
	if err != nil {
		return jsonError(c, err)
	}
	if item = getScript(name); item == nil {
		return jsonError(c, Lang(DefLang, `erropen`, name))
	}
	if item.Settings.Unrun {
		return jsonError(c, Lang(DefLang, `errnorun`, name))
	}
	if err = AddHistoryRun(c.(*Auth).User.ID, name); err != nil {
		return jsonError(c, err)
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
	header := script.Header{
		Name:         name,
		Title:        title,
		AssetsDir:    cfg.AssetsDir,
		LogDir:       cfg.Log.Dir,
		CDN:          cfg.CDN,
		Console:      console,
		IsPlayground: cfg.playground,
		IP:           c.RealIP(),
		UserID:       c.(*Auth).User.ID,
		Constants:    storage.Settings.Constants,
		Lang:         langCode,
		TaskID:       lib.RndNum(),
		ServerPort:   cfg.HTTP.Port,
		HTTP: &lib.HTTPConfig{
			Host:   cfg.HTTP.Host,
			Port:   port,
			Open:   open,
			Theme:  cfg.HTTP.Theme,
			Access: cfg.HTTP.Access,
		},
	}
	if header.IsPlayground {
		header.Playground = &cfg.Playground
		tasksLimit := cfg.Playground.Tasks
		for _, item := range tasks {
			if item.Status < TaskFinished {
				tasksLimit--
			}
		}
		if tasksLimit <= 0 {
			return jsonError(c, Lang(GetLangId(c.(*Auth).User), `errtasklimit`, cfg.Playground.Tasks))
		}
	}
	if src, err = GenSource(item, &header); err != nil {
		return jsonError(c, err)
	}
	if storage.Settings.IncludeSrc {
		if header.SourceCode, err = lib.GzipCompress([]byte(src)); err != nil {
			return jsonError(c, err)
		}
	}
	data, err := script.Encode(header, src)
	if err != nil {
		return jsonError(c, err)
	}
	if storage.Trial.Mode == TrialOn {
		now := time.Now()
		if storage.Trial.Last.Day() != now.Day() {
			storage.Trial.Count++
			storage.Trial.Last = now
			if storage.Trial.Count > TrialDays {
				storage.Trial.Mode = TrialDisabled
				SetActive(false)
			}
			if err = SaveStorage(); err != nil {
				return jsonError(c, err)
			}
		}
	}
	if err = NewTask(header); err != nil {
		return jsonError(c, err)
	}
	if console {
		return c.Blob(http.StatusOK, ``, data.Bytes())
	}
	return c.JSON(http.StatusOK, RunResponse{Success: true, Port: header.HTTP.Port, ID: header.TaskID})
}

func pingHandle(c echo.Context) error {
	return c.HTML(http.StatusOK, Success)
}

func taskStatusHandle(c echo.Context) error {
	var (
		taskStatus TaskStatus
		err        error
		finish     string
	)
	if err = c.Bind(&taskStatus); err != nil {
		return jsonError(c, err)
	}
	if taskStatus.Time != 0 {
		finish = time.Unix(taskStatus.Time, 0).Format(TimeFormat)
	}
	cmd := WsCmd{
		TaskID:  taskStatus.TaskID,
		Cmd:     WcStatus,
		Status:  taskStatus.Status,
		Message: taskStatus.Message,
		Time:    finish,
	}
	if taskStatus.Status == TaskActive {
		task := tasks[taskStatus.TaskID]
		cmd.Task = &Task{
			ID:         task.ID,
			Status:     task.Status,
			Name:       task.Name,
			StartTime:  task.StartTime,
			FinishTime: task.FinishTime,
			UserID:     task.UserID,
			Port:       task.Port,
		}
	}

	for id, client := range clients {
		err := client.Conn.WriteJSON(cmd)
		if err != nil {
			client.Conn.Close()
			delete(clients, id)
		}
	}
	if ptask := tasks[taskStatus.TaskID]; ptask != nil {
		ptask.Status = taskStatus.Status
		if taskStatus.Status >= TaskFinished {
			ptask.Message = taskStatus.Message
			ptask.FinishTime = taskStatus.Time
			if err = SaveTrace(ptask); err != nil {
				return jsonError(c, err)
			}
		}
	}
	return jsonSuccess(c)
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

	for _, item := range tasks {
		if item.ID == uint32(taskid) {
			url := fmt.Sprintf("http://%s:%d/sys?cmd=%s&taskid=%d", Localhost, item.Port, cmd, taskid)
			go func() {
				resp, err := http.Get(url)
				if err == nil {
					resp.Body.Close()
				}
			}()
			break
		}
	}
	return jsonSuccess(c)
}

func tasksHandle(c echo.Context) error {
	list := ListTasks()
	/*	for i := len(list)/2 - 1; i >= 0; i-- {
		opp := len(list) - 1 - i
		list[i], list[opp] = list[opp], list[i]
	}*/
	listInfo := make([]TaskInfo, len(list))
	for i, item := range list {
		var finish string
		if item.FinishTime > 0 {
			finish = time.Unix(item.FinishTime, 0).Format(TimeFormat)
		}
		listInfo[i] = TaskInfo{
			ID:         item.ID,
			Status:     item.Status,
			Name:       item.Name,
			StartTime:  time.Unix(item.StartTime, 0).Format(TimeFormat),
			FinishTime: finish,
			UserID:     item.UserID,
			Port:       item.Port,
			Message:    item.Message,
		}
	}
	return c.JSON(http.StatusOK, &TasksResponse{
		List: listInfo,
	})
}

func removeTaskHandle(c echo.Context) error {
	idTask, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if _, ok := tasks[uint32(idTask)]; !ok {
		return jsonError(c, fmt.Errorf(`task %d has not been found`, idTask))
	}
	delete(tasks, uint32(idTask))
	RemoveTask(uint32(idTask))
	return tasksHandle(c)
}

func trialHandle(c echo.Context) error {
	var (
		err  error
		mode int
	)
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
		if err = SetActive(storage.Trial.Mode == TrialOn); err != nil {
			return jsonError(c, err)
		}
		if err = SaveStorage(); err != nil {
			return jsonError(c, err)
		}
	}
	return proSettingsHandle(c)
}
