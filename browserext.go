// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/users"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	ChromeExtension = ``
)

type ExtScript struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type ExtRun struct {
	Name string `json:"name"`
	Open bool   `json:"open"`
	URL  string `json:"url"`
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
	/*	ip := c.RealIP()
		host := c.Request().Host
		if offPort := strings.LastIndex(c.Request().Host, `:`); offPort > 0 {
			host = host[:offPort]
		}
		if !lib.IsLocalhost(host, ip) {
			return AccessDenied(http.StatusForbidden)
		}*/
	var ext ExtInfo
	if err = c.Bind(&ext); err != nil {
		return jsonError(c, err)
	}

	list := []ExtScript{
		{`welcome`, ext.Url},
		{`my.scrypt`, `Скрипт для запуска`},
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
	var (
		err error
		ext ExtRun
	)
	if err = c.Bind(&ext); err != nil {
		return jsonError(c, err)
	}

	user := c.(*Auth).User
	rs := RunScript{
		Name: ext.Name,
		Open: ext.Open && cfg.HTTP.Host == Localhost,
		User: *user,
		/*		users.User{
				ID:       users.XAdminID,
				Nickname: users.RootUser,
				RoleID:   users.BrowserID,
			},*/
		Role: users.Role{
			ID:   users.BrowserID,
			Name: users.BrowserRole,
		},
		IP: Localhost,
	}
	if err = systemRun(&rs); err != nil {
		NewNotification(&Notification{
			Text:   fmt.Sprintf(`Browser extension error: %s`, err.Error()),
			UserID: user.ID,
			RoleID: users.BrowserID,
			Script: rs.Name,
		})
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, &RunResponse{Success: true, Port: rs.Port, ID: rs.ID})
}
