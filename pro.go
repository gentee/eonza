// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/users"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type ProOptions struct {
	Active   bool              `json:"active"`
	Settings users.ProSettings `json:"settings"`
	Storage  StorageResponse   `json:"storage"`
}

const (
	Pro = true
)

func IsProActive() bool {
	return true
}

func CheckAdmin(c echo.Context) error {
	return AdminAccess(c.(*Auth).User.ID)
}

func GetUserRole(id, idrole uint32) (uname string, rname string) {
	if idrole >= users.ResRoleID {
		uname, rname = GetSchedulerName(id, idrole)
	} else {
		uname, rname = ProGetUserRole(id)
	}
	if len(uname) == 0 {
		uname = fmt.Sprintf("%x", id)
	}
	if len(rname) == 0 {
		rname = fmt.Sprintf("%x", idrole)
	}
	return
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

func GetTrialMode() int {
	return storage.Trial.Mode
}

func TaskCheck(taskID uint32, userID uint32) (bool, error) {
	var status int

	if v, ok := tasks[taskID]; ok && v.UserID == userID {
		status = v.Status
		if v.Status == TaskActive || v.Status == TaskWaiting ||
			(v.Status == TaskFinished && time.Now().Unix() <= v.FinishTime+1) {
			return v.Status < TaskFinished, nil
		}
	}
	return status < TaskFinished, fmt.Errorf(`access denied task %d / user %d`, taskID, userID)
}

func ProInit(psw []byte, counter uint32) {
	CallbackPassCounter = StoragePassCounter
	CallbackTitle = GetTitle
	CallbackTrial = GetTrialMode
	CallbackTaskCheck = TaskCheck
	LoadPro(psw, counter, cfg.path, cfg.Users.Dir)
}

func proSettingsHandle(c echo.Context) error {
	var response ProOptions

	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	response.Active = IsProActive()
	response.Settings = ProSettings()
	response.Storage = PassStorage()

	return c.JSON(http.StatusOK, &response)
}
