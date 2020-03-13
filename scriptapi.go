// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type ScriptResponse struct {
	Script
	original string       `json:"original"`
	History  []ScriptItem `json:"history,omitempty"`
	Error    string       `json:"error,omitempty"`
}

func getScriptHandle(c echo.Context) error {
	var response ScriptResponse

	name := c.QueryParam(`name`)
	if len(name) == 0 {
		name = LatestHistoryEditor(c.(*Auth).User.ID)
		if len(name) == 0 {
			name = `new`
		}
	}
	script := scripts[name]
	if script == nil {
		response.Error = Lang(`erropen`, name)
	} else {
		response.Script = *script
		if response.Script.Settings.Name == `new` {
			response.Script.Settings.Name = lib.UniqueName(7)
			response.Script.Settings.Title = Lang(`newscript`)
		} else {
			AddHistoryEditor(c.(*Auth).User.ID, script.Settings.Name)
			response.original = name
		}
		response.History = GetHistoryEditor(c.(*Auth).User.ID)
	}
	return c.JSON(http.StatusOK, &response)
}

func saveScriptHandle(c echo.Context) error {
	var (
		script ScriptResponse
		err    error
	)
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = c.Bind(&script); err != nil {
		return errResult()
	}
	if err = script.Validate(); err != nil {
		return errResult()
	}
	if len(script.original) == 0 {
		if err = AddHistoryEditor(c.(*Auth).User.ID, script.Settings.Name); err != nil {
			return errResult()
		}
	}
	if err = (&script.Script).SaveScript(script.original); err != nil {
		return errResult()
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}
