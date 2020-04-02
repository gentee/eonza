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

func jsonError(c echo.Context, err error) error {
	return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
}

func runHandle(c echo.Context) error {
	var response Response

	name := c.QueryParam(`name`)
	port, err := getPort()
	if err != nil {
		response.Error = fmt.Sprint(err)
	} else if _, ok := scripts[name]; !ok {
		response.Error = Lang(`erropen`, name)
	} else if err := script.Encode(script.Header{
		Name:       name,
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
		response.Error = fmt.Sprint(err)
	} else {
		response.Success = true
	}
	return c.JSON(http.StatusOK, response)
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
