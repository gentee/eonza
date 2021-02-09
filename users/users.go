// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package users

const (
	RootUser = `root`
	RootID   = 0 // Don't change. Must be zero
	RootRole = `admin`
)

type Role struct {
	ID       uint32
	Name     string
	Allow    string
	Disallow string
	//	Scripts       int
	Tasks         int
	Notifications int
	//	Settings      int
	//	Pro           int
}

type User struct {
	ID           uint32
	Nickname     string
	PasswordHash []byte
	Role         uint32
}

var (
	Users map[uint32]User
	Roles []Role
)

func InitRoot(psw []byte) {
	Roles = []Role{
		{ID: RootID, Name: RootRole},
	}
	Users = map[uint32]User{
		RootID: {ID: RootID, Nickname: RootUser, PasswordHash: psw, Role: RootID},
	}
}
