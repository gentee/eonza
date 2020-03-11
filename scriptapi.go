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
	IsNew bool   `json:"isnew"`
	Error string `json:"error,omitempty"`
}

/*
type ScriptRequest struct {
	Script Script `json:"script"`
	IsNew  bool   `json:"isnew"`
}*/

func getScriptHandle(c echo.Context) error {
	var response ScriptResponse

	name := c.QueryParam(`name`)
	if len(name) == 0 {
		name = LatestHistory(c.(*Auth).User.ID, HistEditor)
		if len(name) == 0 {
			name = `new`
		}
	}
	script := GetScript(name)
	if script == nil {
		response.Error = Lang(`erropen`, name)
	} else {
		response.Script = *script
		if response.Script.Settings.Name == `new` {
			response.Script.Settings.Name = lib.UniqueName(7)
			response.Script.Settings.Title = Lang(`newscript`)
			response.IsNew = true
		} else {
			AddHistory(c.(*Auth).User.ID, HistEditor, script.Settings.Name)
		}
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
	if script.IsNew {
		if err = AddHistory(c.(*Auth).User.ID, HistEditor, script.Settings.Name); err != nil {
			return errResult()
		}
	}
	if err = SaveScript(script.Script); err != nil {
		return errResult()
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}
