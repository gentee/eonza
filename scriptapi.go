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
	Name     string        `json:"name"`
	Title    string        `json:"title"`
	Desc     string        `json:"desc,omitempty"`
	Unrun    bool          `json:"unrun,omitempty"`
	Embedded bool          `json:"embedded,omitempty"`
	Folder   bool          `json:"folder,omitempty"`
	Params   []scriptParam `json:"params,omitempty"`
}

type ScriptResponse struct {
	Script
	Original string       `json:"original"`
	History  []ScriptItem `json:"history,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type ListResponse struct {
	Map   map[string]ScriptItem `json:"map,omitempty"`
	List  []ScriptItem          `json:"list,omitempty"`
	Cache int32                 `json:"cache"`
	Error string                `json:"error,omitempty"`
}

var (
	hotVersion int32 = 1
)

func deleteScriptHandle(c echo.Context) error {
	var response Response

	if err := DeleteScript(c.QueryParam(`name`)); err != nil {
		response.Error = fmt.Sprint(err)
	} else {
		response.Success = true
		hotVersion++
	}
	return c.JSON(http.StatusOK, response)
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
	hotVersion++
	return c.JSON(http.StatusOK, Response{Success: true})
}

func ScriptToItem(script *Script) ScriptItem {
	return ScriptItem{
		Name:     script.Settings.Name,
		Title:    script.Settings.Title,
		Desc:     script.Settings.Desc,
		Unrun:    script.Settings.Unrun,
		Embedded: script.embedded,
		Folder:   script.folder,
		Params:   script.Params,
	}

}

func listScriptHandle(c echo.Context) error {
	resp := &ListResponse{
		Cache: hotVersion,
	}

	if c.QueryParam(`cache`) != fmt.Sprint(hotVersion) {
		list := make(map[string]ScriptItem)

		for key, item := range scripts {
			list[key] = ScriptToItem(item)
		}
		resp.Map = list
	}
	return c.JSON(http.StatusOK, resp)
}

func listRunHandle(c echo.Context) error {
	list := make([]ScriptItem, 0)
	userId := c.(*Auth).User.ID
	if _, ok := userSettings[userId]; !ok {
		return jsonError(c, Lang(`unknownuser`, userId))
	}
	for _, name := range userSettings[userId].History.Run {
		if item, ok := scripts[name]; ok {
			list = append(list, ScriptToItem(item))
		}
	}
	return c.JSON(http.StatusOK, &ListResponse{
		List: list,
	})
}
