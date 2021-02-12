// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build !pro

package main

import (
	"eonza/users"
	"fmt"

	"github.com/labstack/echo/v4"
)

const Pro = false

var (
	Users map[uint32]users.User
	Roles map[uint32]users.Role
)

func ProInit(psw []byte) {
	Roles, Users = users.InitUsers(psw)
}

func GetUser(id uint32) (user users.User, ok bool) {
	user, ok = Users[id]
	return
}

func GetUsers() []users.User {
	user := Users[users.XRootID]
	return []users.User{
		user,
	}
}

func SetActive(active bool) error {
	return nil
}

func SetUserPassword(id uint32, hash []byte) error {
	if user, ok := GetUser(id); ok {
		user.PasswordHash = hash
		Users[id] = user
	}
	return nil
}

func proSettingsHandle(c echo.Context) error {
	return jsonError(c, fmt.Errorf(`Unsupported`))
}

func ProApi(e *echo.Echo) {
}
