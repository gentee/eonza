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

type ScriptItem struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Desc     string `json:"desc,omitempty"`
	Unrun    bool   `json:"unrun,omitempty"`
	Embedded bool   `json:"embedded,omitempty"`
}

type ScriptResponse struct {
	Script
	Original string       `json:"original"`
	History  []ScriptItem `json:"history,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type ListResponse struct {
	List  map[string]ScriptItem `json:"list"`
	Error string                `json:"error,omitempty"`
}

func deleteScriptHandle(c echo.Context) error {
	var response ScriptResponse

	if err := DeleteScript(c.QueryParam(`name`)); err != nil {
		response.Error = fmt.Sprint(err)
	}
	return c.JSON(http.StatusOK, &response)
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
			response.Original = name
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
	if len(script.Original) == 0 {
		if err = AddHistoryEditor(c.(*Auth).User.ID, script.Settings.Name); err != nil {
			return errResult()
		}
	}
	if err = (&script.Script).SaveScript(script.Original); err != nil {
		return errResult()
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}

func listScriptHandle(c echo.Context) error {
	list := make(map[string]ScriptItem)

	for key, item := range scripts {
		list[key] = ScriptItem{
			Name:     key,
			Title:    item.Settings.Title,
			Desc:     item.Settings.Desc,
			Unrun:    item.Settings.Unrun,
			Embedded: item.embedded,
		}
	}
	return c.JSON(http.StatusOK, &ListResponse{
		List: list})
}
