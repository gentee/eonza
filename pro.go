// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

//go:build pro

package main

import (
	"eonza/users"
	"fmt"
	"net/http"
	"time"

	pro "github.com/gentee/eonza-pro"
	"github.com/labstack/echo/v4"
)

type ProOptions struct {
	Active   bool                `json:"active"`
	License  users.LicenseInfo   `json:"license"`
	Settings users.ProSettings   `json:"settings"`
	Storage  pro.StorageResponse `json:"storage"`
	Trial    Trial               `json:"trial"`
}

const (
	Pro = true
)

func Licensed() bool {
	return pro.Licensed()
}

func IsProActive() bool {
	return pro.Active
}

func SetActive() {
	pro.SetActive()
}

func VerifyKey() {
	pro.VerifyKey(false)
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

func IsDecrypted() bool {
	return pro.IsDecrypted()
}

func IsAutoFill() bool {
	return pro.IsAutoFill()
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

func GetTrialMode() int {
	return storage.Trial.Mode
}

func TaskCheck(taskID uint32, userID uint32) error {
	if v, ok := tasks[taskID]; ok && v.UserID == userID {
		if v.Status == TaskActive || v.Status == TaskWaiting ||
			(v.Status == TaskFinished && time.Now().Unix() <= v.FinishTime+1) {
			return nil
		}
	}
	return fmt.Errorf(`access denied task %d / user %d`, taskID, userID)
}

func ProInit(psw []byte, counter uint32) {
	pro.CallbackPassCounter = StoragePassCounter
	pro.CallbackTitle = GetTitle
	pro.CallbackTrial = GetTrialMode
	pro.CallbackTaskCheck = TaskCheck
	pro.LoadPro(psw, counter, cfg.path, cfg.Users.Dir)
}

func proSettingsHandle(c echo.Context) error {
	var response ProOptions

	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	response.Active = IsProActive()
	response.License = pro.GetLicenseInfo()
	response.Trial = storage.Trial
	response.Settings = pro.Settings()
	response.Storage = pro.PassStorage()

	return c.JSON(http.StatusOK, &response)
}

func SecureConstants() map[string]string {
	return pro.SecureConstants()
}

func ProApi(e *echo.Echo) {
	pro.ProApi(e)
}
