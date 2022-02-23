// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"eonza/users"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	es "eonza/script"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
	md "github.com/labstack/echo/v4/middleware"
)

func LocalAuthHandle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		var (
			ok     bool
			userID uint32
			user   users.User
		)
		ip := c.RealIP()
		host := c.Request().Host
		if offPort := strings.LastIndex(c.Request().Host, `:`); offPort > 0 {
			host = host[:offPort]
		}
		if !lib.IsLocalhost(host, ip) {
			return AccessDenied(http.StatusForbidden)
		}
		lang := LangDefCode
		if IsScript {
			user = scriptTask.Header.User
			lang = scriptTask.Header.Lang
		} else {
			userID = uint32(users.XRootID)
			if user, ok = GetUser(userID); !ok {
				return AccessDenied(http.StatusUnauthorized)
			}
			if u, ok := userSettings[user.ID]; ok {
				lang = u.Lang
			}
		}
		auth := &Auth{
			Context: c,
			User:    &user,
			Lang:    lang,
		}
		err = next(auth)
		return
	}
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
			RoleID:     task.RoleID,
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

func notificationHandle(c echo.Context) error {
	var (
		postNfy es.PostNfy
		err     error
	)
	if err = c.Bind(&postNfy); err != nil {
		return jsonError(c, err)
	}
	nfy := Notification{
		Text:   postNfy.Text,
		UserID: users.XRootID,
		RoleID: users.XAdminID,
		Script: postNfy.Script,
	}
	if ptask, ok := tasks[postNfy.TaskID]; ok {
		nfy.UserID = ptask.UserID
		nfy.RoleID = ptask.RoleID
	}
	if err = NewNotification(&nfy); err != nil {
		return jsonError(c, err)
	}
	return jsonSuccess(c)
}

func runScriptHandle(c echo.Context) error {
	var (
		postScript es.PostScript
		err        error
	)
	if err = c.Bind(&postScript); err != nil {
		return jsonError(c, err)
	}
	if tasks[postScript.TaskID] == nil {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}
	rs := RunScript{
		Name:    postScript.Script,
		Open:    !postScript.Silent,
		Console: false,
		Data:    postScript.Data,
		User: users.User{
			ID:       postScript.TaskID,
			Nickname: GetTaskName(postScript.TaskID),
			RoleID:   users.ScriptsID,
		},
		Role: users.Role{
			ID:   users.ScriptsID,
			Name: users.ScriptsRole,
		},
		IP: c.RealIP(),
	}
	if err := systemRun(&rs); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, RunResponse{Success: true, Port: rs.Port, ID: rs.ID})
}

func RunLocalServer(port int) *echo.Echo {
	e := echo.New()

	e.HideBanner = true
	e.Use(LocalAuthHandle)
	e.Use(Logger)
	e.Use(md.Recover())

	e.GET("/ping", pingHandle)
	if IsScript {
		e.GET("/info", infoHandle)
		e.GET("/sys", sysHandle)
		es.CmdServer(e)
	} else {
		e.GET("/api/run", runHandle)
		e.GET("/api/randid", randidHandle)
		e.POST("/api/event", eventHandle)
		e.POST("/api/notification", notificationHandle)
		e.POST("/api/taskstatus", taskStatusHandle)
		e.POST("/api/runscript", runScriptHandle)
		e.POST("/api/extqueue", extQueueHandle)
	}
	go func() {
		if IsScript {
			e.Logger.SetOutput(io.Discard)
		}
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil && !isShutdown {
			golog.Fatal(err)
		}
	}()
	return e
}
