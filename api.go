// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"

	"eonza/lib"
	"eonza/script"

	"github.com/labstack/echo/v4"
)

type TaskStatus struct {
	TaskID  uint32 `json:"taskid"`
	Status  int    `json:"status"`
	Message string `json:"msg,omitempty"`
	Time    uint32 `json:"time,omitempty"`
}

func jsonError(c echo.Context, err interface{}) error {
	return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
}

func jsonSuccess(c echo.Context) error {
	return c.JSON(http.StatusOK, Response{Success: true})
}

func runHandle(c echo.Context) error {
	var (
		item *Script
		ok   bool
	)
	name := c.QueryParam(`name`)
	port, err := getPort()
	if err != nil {
		return jsonError(c, err)
	}
	if item, ok = scripts[name]; !ok {
		return jsonError(c, Lang(`erropen`, name))
	}
	if err = AddHistoryRun(c.(*Auth).User.ID, name); err != nil {
		return jsonError(c, err)
	}
	if err := script.Encode(script.Header{
		Name:       name,
		Title:      item.Settings.Title,
		AssetsDir:  cfg.AssetsDir,
		UserID:     c.(*Auth).User.ID,
		TaskID:     lib.RndNum(),
		ServerPort: cfg.HTTP.Port,
		HTTP: &lib.HTTPConfig{
			Port:  port,
			Open:  true,
			Theme: cfg.HTTP.Theme,
		},
	}); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}

func pingHandle(c echo.Context) error {
	return c.HTML(http.StatusOK, Success)
}

func taskStatusHandle(c echo.Context) error {
	var taskStatus TaskStatus

	if err := c.Bind(&taskStatus); err != nil {
		return jsonError(c, err)
	}
	switch taskStatus.Status {
	case TaskActive:
		//		usePort(taskStatus.Number)
		fmt.Println(`ports`, ports[:16])
	case TaskFailed:
		//		ports[]
	}
	return c.JSON(http.StatusOK, Response{
		Success: true,
	})
}
