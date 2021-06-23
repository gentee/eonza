// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"eonza/users"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	ChromeExtension = `lnhmpeahpfhnpijjccofkfapmadmefih`
)

type ExtScript struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type ExtListResponse struct {
	List  []ExtScript `json:"list,omitempty"`
	Error string      `json:"error,omitempty"`
}

type ExtInfo struct {
	Url string `json:"url"`
}

func browserExtHandle(c echo.Context) error {
	var err error
	// Now it supports only localhost
	// all hosts -> remove api/browserext from auth
	ip := c.RealIP()
	host := c.Request().Host
	if offPort := strings.LastIndex(c.Request().Host, `:`); offPort > 0 {
		host = host[:offPort]
	}
	if !lib.IsLocalhost(host, ip) {
		return AccessDenied(http.StatusForbidden)
	}
	var ext ExtInfo
	if err = c.Bind(&ext); err != nil {
		return jsonError(c, err)
	}

	list := []ExtScript{
		{`ooops`, ext.Url},
		{`my.script`, `Скрипт для запуска`},
	}
	//	list := make([]ExtScript, 0)
	/*	userId := c.(*Auth).User.ID
		if _, ok := userSettings[userId]; ok {
			for _, name := range userSettings[userId].History.Run {
				if item := getScript(name); item != nil {
					list = append(list, ScriptToItem(c, item))
				}
			}
		}*/
	return c.JSON(http.StatusOK, &ExtListResponse{
		List: list,
	})
}

func browserRunHandle(c echo.Context) error {
	rs := RunScript{
		Name: `welcome`,
		User: users.User{
			ID:       users.XAdminID,
			Nickname: users.RootUser,
			RoleID:   users.BrowserID,
		},
		Role: users.Role{
			ID:   users.BrowserID,
			Name: users.BrowserRole,
		},
		IP: Localhost,
	}
	if err := systemRun(&rs); err != nil {
		NewNotification(&Notification{
			Text:   fmt.Sprintf(`Browser extension error: %s`, err.Error()),
			UserID: users.XAdminID,
			RoleID: users.BrowserID,
			Script: rs.Name,
		})
	}
	return c.JSON(http.StatusOK, &Response{
		Success: true,
	})
}
