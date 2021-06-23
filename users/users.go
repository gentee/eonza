// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package users

import (
	"path"
	"regexp"
	"strings"

	echo "github.com/labstack/echo/v4"
)

const (
	RootUser    = `root`
	RootRole    = `admin`
	TimersRole  = `timers`
	EventsRole  = `events`
	ScriptsRole = `scripts`
	BrowserRole = `browser`
	ResRoleID   = 0xffffff00
	BrowserID   = 0xfffffffc
	ScriptsID   = 0xfffffffd
	EventsID    = 0xfffffffe
	TimersID    = 0xffffffff
	XRootID     = 1
	XAdminID    = 1
)

type LicenseInfo struct {
	Status  int    `json:"status"`
	License string `json:"license"`
	Volume  int    `json:"volume"`
	Expire  string `json:"expire"`
}

type ProSettings struct {
	Twofa  bool   `json:"twofa"`
	Master string `json:"master"`
}

type Role struct {
	ID            uint32 `json:"id"`
	Name          string `json:"name"`
	Allow         string `json:"allow"`
	Tasks         int    `json:"tasks"`
	Notifications int    `json:"notifications"`

	patterns []string
	regex    []*regexp.Regexp
	//	Disallow string
	//	Scripts       int
	//	Settings      int
	//	Pro           int
}

type User struct {
	ID           uint32 `json:"id"`
	RoleID       uint32 `json:"roleid"`
	PassCounter  uint32 `json:"-"`
	Nickname     string `json:"nickname"`
	PasswordHash []byte `json:"-"`
}

type Auth struct {
	echo.Context
	User *User
	Lang string
}

func InitUsers(psw []byte, counter uint32) (map[uint32]Role, map[uint32]User) {
	Roles := map[uint32]Role{
		XAdminID:  {ID: XAdminID, Name: RootRole},
		TimersID:  {ID: TimersID, Name: TimersRole},
		EventsID:  {ID: EventsID, Name: EventsRole},
		ScriptsID: {ID: ScriptsID, Name: ScriptsRole},
		BrowserID: {ID: BrowserID, Name: BrowserRole},
	}
	Users := map[uint32]User{
		XRootID: {ID: XRootID, Nickname: RootUser, PasswordHash: psw, RoleID: XAdminID,
			PassCounter: counter},
	}
	return Roles, Users
}

func ParseAllow(role Role) Role {
	role.patterns = nil
	role.regex = nil
	items := strings.Split(strings.ReplaceAll(role.Allow, "\n", " "), " ")
	for _, item := range items {
		item = strings.TrimSpace(item)
		ilen := len(item)
		if ilen > 0 {
			if item[0] == '/' && item[ilen-1] == '/' {
				if re, err := regexp.Compile(item[1 : ilen-1]); err == nil {
					role.regex = append(role.regex, re)
				}
			} else {
				role.patterns = append(role.patterns, item)
			}
		}
	}
	return role
}

func MatchAllow(name, ipath string, role Role) bool {
	for _, pattern := range role.patterns {
		if name == pattern {
			return true
		}
	}
	for _, re := range role.regex {
		if re.MatchString(path.Join(ipath, name)) {
			return true
		}
	}
	return false
}

/*
func NewUser(nickname string) (uint32, error) {
	user := User{
		Nickname: nickname,
	}
	if !lib.ValidateSysName(nickname) {
		return 0, fmt.Errorf(Lang(DefLang, `invalidfield`), Lang(DefLang, `nickname`))
	}
	for _, item := range storage.Users {
		if item.Nickname == nickname {
			return 0, fmt.Errorf(Lang(DefLang, `errnickname`), nickname)
		}

	}
	private, public, err := lib.GenerateKeys()
	if err != nil {
		return 0, err
	}
	user.PublicKey = public
	user.ID = crc32.ChecksumIEEE(private)
	if err = os.WriteFile(filepath.Join(cfg.Users.Dir, user.Nickname+`.key`),
		[]byte(hex.EncodeToString(private)), 0777 os.ModePerm); err != nil {
		return 0, err
	}
	storage.Users[user.ID] = &user
	userSettings[user.ID] = UserSettings{
		ID:   user.ID,
		Lang: appInfo.Lang,
	}
	return user.ID, nil
}
*/
