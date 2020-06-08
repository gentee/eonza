// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Options struct {
	Common Settings     `json:"common"`
	User   UserSettings `json:"user"`
}

func settingsHandle(c echo.Context) error {
	var response Options

	response.Common = storage.Settings
	response.User = userSettings[c.(*Auth).User.ID]
	return c.JSON(http.StatusOK, &response)
}

func saveSettingsHandle(c echo.Context) error {
	var (
		options Options
		err     error
	)
	if err = c.Bind(&options); err != nil {
		return jsonError(c, err)
	}
	storage.Settings = options.Common
	if err = SaveStorage(); err != nil {
		return jsonError(c, err)
	}
	id := c.(*Auth).User.ID
	user := userSettings[id]
	user.Lang = options.User.Lang
	userSettings[id] = user

	if err = SaveUser(id); err != nil {
		return jsonError(c, err)
	}
	return jsonSuccess(c)
}
