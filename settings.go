// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/users"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type Options struct {
	Common Settings     `json:"common"`
	User   UserSettings `json:"user"`
}

type Psw struct {
	CurPassword string `json:"curpassword"`
	Password    string `json:"password"`
}

func settingsHandle(c echo.Context) error {
	var response Options

	response.Common = storage.Settings
	response.User = userSettings[c.(*Auth).User.ID]
	return c.JSON(http.StatusOK, &response)
}

func saveSettingsHandle(c echo.Context) error {
	var (
		options  Options
		err      error
		hideTray bool
	)
	if err = c.Bind(&options); err != nil {
		return jsonError(c, err)
	}
	hideTray = storage.Settings.HideTray
	storage.Settings = options.Common
	if err = SaveStorage(); err != nil {
		return jsonError(c, err)
	}
	id := c.(*Auth).User.ID
	user := userSettings[id]
	user.Lang = options.User.Lang
	userSettings[id] = user
	if isTray && !hideTray && storage.Settings.HideTray {
		HideTray()
	}
	if err = SaveUser(id); err != nil {
		return jsonError(c, err)
	}
	return jsonSuccess(c)
}

func setPasswordHandle(c echo.Context) error {
	var (
		psw  Psw
		err  error
		hash []byte
	)
	user := c.(*Auth).User
	if cfg.playground {
		return jsonError(c, Lang(GetLangId(user), `errplaypsw`))
	}
	if err = c.Bind(&psw); err != nil {
		return jsonError(c, err)
	}
	if len(user.PasswordHash) > 0 {
		err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(psw.CurPassword))
		if err != nil {
			return jsonError(c, Lang(GetLangId(user), `invalidpsw`))
		}
	}
	if len(psw.Password) > 0 {
		hash, err = bcrypt.GenerateFromPassword([]byte(psw.Password), 11)
		if err != nil {
			return jsonError(c, err)
		}
	}
	if err = SetUserPassword(user.ID, hash); err != nil {
		return jsonError(c, err)
	}
	if user.ID == users.XRootID {
		storage.Settings.PasswordHash = hash
		storage.PassCounter++
		if err = SaveStorage(); err != nil {
			return jsonError(c, err)
		}
	}
	return jsonSuccess(c)
}

func saveFavsHandle(c echo.Context) error {
	var (
		favs []Fav
		err  error
	)
	if err = c.Bind(&favs); err != nil {
		return jsonError(c, err)
	}
	id := c.(*Auth).User.ID
	user := userSettings[id]
	user.Favs = favs
	userSettings[id] = user
	if err = SaveUser(id); err != nil {
		return jsonError(c, err)
	}
	return jsonSuccess(c)
}
