// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/users"
	"fmt"
	"strconv"

	"github.com/labstack/echo/v4"
)

func showTaskHandle(c echo.Context) error {

	idtask, _ := strconv.ParseUint(c.Param(`id`), 10, 32)
	ptask := tasks[uint32(idtask)]
	if ptask == nil {
		return jsonError(c, fmt.Errorf(`task %d has not been found`, idtask))
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
			return jsonError(c, fmt.Errorf(`Access denied`))
		}
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
