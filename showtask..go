// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/script"
	"eonza/users"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

func showTaskAccess(c echo.Context, task string) (*Task, *users.User, error) {
	idtask, _ := strconv.ParseUint(task, 10, 32)
	ptask := tasks[uint32(idtask)]
	if ptask == nil {
		return nil, nil, jsonError(c, fmt.Errorf(`task %d has not been found`, idtask))
	}
	user := c.(*Auth).User
	if user.RoleID != users.XAdminID {
		var access bool
		if role, ok := GetRole(user.RoleID); ok {
			taskFlag := role.Tasks
			access = (taskFlag&4 == 4) ||
				(taskFlag&1 == 1 && user.ID == ptask.UserID) ||
				(taskFlag&2 == 2 && user.RoleID == ptask.RoleID)
		}
		if !access {
			return nil, nil, jsonError(c, fmt.Errorf(`Access denied`))
		}
	}
	return ptask, user, nil
}

func showTaskHandle(c echo.Context) error {
	var (
		err   error
		ptask *Task
		user  *users.User
	)
	if ptask, user, err = showTaskAccess(c, c.Param(`id`)); err != nil {
		return err
	}
	if item := getScript(ptask.Name); item != nil {
		c.Set(`Title`, ScriptLang(item, GetLangCode(user), item.Settings.Title))
	} else {
		c.Set(`Title`, ptask.Name)
	}
	c.Set(`Task`, ptask)
	c.Set(`tpl`, `script`)
	return indexHandle(c)
}

func saveReportHandle(c echo.Context) error {
	var (
		err   error
		ptask *Task
	)
	if ptask, _, err = showTaskAccess(c, c.QueryParam(`taskid`)); err != nil {
		return err
	}
	reportid, err := strconv.ParseUint(c.QueryParam(`reportid`), 10, 32)
	if err != nil {
		return jsonError(c, err)
	}
	_, replist := GetTaskFiles(ptask.ID, false)
	if int(reportid) >= len(replist) {
		return jsonError(c, fmt.Errorf(`report %d has not been found`, reportid))
	}
	rep := replist[reportid]
	name := rep.Title
	for _, s := range []string{`/`, `\`, `:`} {
		name = strings.ReplaceAll(name, s, `_`)
	}
	ext := script.GetReportExt(rep)
	c.Response().Header().Set(echo.HeaderContentDisposition,
		fmt.Sprintf("attachment; filename=%s.%s", name, ext))
	return c.Blob(http.StatusOK, fmt.Sprintf("text/%s", ext), []byte(rep.Body))
}
