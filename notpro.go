// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build !pro

package main

import (
	"eonza/users"
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
)

const Pro = false

var (
	Users    map[uint32]users.User
	Roles    map[uint32]users.Role
	proMutex = &sync.Mutex{}
)

func IsProActive() bool {
	return false
}

func ProInit(psw []byte, counter uint32) {
	Roles, Users = users.InitUsers(psw, counter)
}

func GetRole(id uint32) (role users.Role, ok bool) {
	role, ok = Roles[id]
	return
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

func CheckAdmin(c echo.Context) error {
	return nil
}

func ScriptAccess(name, ipath string, roleid uint32) error {
	return nil
}

func SetActive(active bool) error {
	return nil
}

func SetUserPassword(id uint32, hash []byte) error {
	proMutex.Lock()
	defer proMutex.Unlock()
	if user, ok := GetUser(id); ok {
		user.PassCounter++
		user.PasswordHash = hash
		Users[id] = user
	}
	return nil
}

func IncPassCounter(id uint32) error {
	proMutex.Lock()
	defer proMutex.Unlock()
	if user, ok := GetUser(id); ok {
		user.PassCounter++
		Users[id] = user
	}
	return nil
}

func proSettingsHandle(c echo.Context) error {
	return jsonError(c, fmt.Errorf(`Unsupported`))
}

func ProApi(e *echo.Echo) {
}
