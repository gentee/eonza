// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build pro

package main

import (
	"eonza/users"
	"fmt"
	"net/http"

	pro "github.com/gentee/eonza-pro"
	"github.com/labstack/echo/v4"
)

type ProOptions struct {
	Active   bool              `json:"active"`
	Settings users.ProSettings `json:"settings"`
	Trial    Trial             `json:"trial"`
}

const (
	Pro = true
)

func IsProActive() bool {
	return pro.Active
}

func SetActive(active bool) error {
	return pro.SetActive(active)
}

func CheckAdmin(c echo.Context) error {
	return pro.AdminAccess(c.(*Auth).User.ID)
}

func ScriptAccess(name, ipath string, roleid uint32) error {
	if roleid >= users.ResRoleID {
		return nil
	}
	return pro.ScriptAccess(name, ipath, roleid)
}

func GetRole(id uint32) (role users.Role, ok bool) {
	return pro.GetRole(id)
}

func GetUser(id uint32) (user users.User, ok bool) {
	return pro.GetUser(id)
}

func GetUserRole(id, idrole uint32) (uname string, rname string) {
	if idrole >= users.ResRoleID {
		uname, rname = GetSchedulerName(id, idrole)
	} else {
		uname, rname = pro.GetUserRole(id)
	}
	if len(uname) == 0 {
		uname = fmt.Sprintf("%x", id)
	}
	if len(rname) == 0 {
		rname = fmt.Sprintf("%x", idrole)
	}
	return
}

func GetUsers() []users.User {
	return pro.GetUsers()
}

func SetUserPassword(id uint32, hash []byte) error {
	return pro.SetUserPassword(id, hash)
}

func IncPassCounter(id uint32) error {
	return pro.IncPassCounter(id)
}

func IsTwofa() bool {
	return pro.IsTwofa()
}

func TwofaQR(id uint32) (string, error) {
	return pro.TwofaQR(id)
}

func ValidateOTP(user users.User, otp string) error {
	return pro.ValidateOTP(user, otp)
}

func GetTitle() string {
	ret := storage.Settings.Title
	if len(ret) == 0 {
		ret = appInfo.Title
	} else {
		ret += `/eonza`
	}
	return ret
}

func ProInit(psw []byte, counter uint32) {
	pro.CallbackPassCounter = StoragePassCounter
	pro.CallbackTitle = GetTitle
	pro.LoadPro(storage.Trial.Mode > TrialOff, psw, counter, cfg.path)
}

func proSettingsHandle(c echo.Context) error {
	var response ProOptions

	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	response.Active = pro.Active
	response.Trial = storage.Trial
	response.Settings = pro.Settings()
	return c.JSON(http.StatusOK, &response)
}

func ProApi(e *echo.Echo) {
	pro.ProApi(e)
}
