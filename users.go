// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"eonza/lib"
	es "eonza/script"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v2"
)

type History struct {
	Editor []string `yaml:"editor"`
	Run    []string `yaml:"run"`
}

// UserSettings stores the user's settings
type UserSettings struct {
	ID      uint32  `json:"id" yaml:"id"`
	Lang    string  `json:"lang" yaml:"lang"`
	History History `json:"history" yaml:"history"`
}

// User stores user's parameters
type User struct {
	ID        uint32
	Nickname  string
	PublicKey []byte
}

var (
	userSettings = make(map[uint32]UserSettings)
)

func LoadUsers() error {
	var err error
	for _, item := range storage.Users {
		userSettings[item.ID] = UserSettings{
			ID:   item.ID,
			Lang: appInfo.Lang,
		}
	}

	err = filepath.Walk(cfg.Users.Dir, func(path string, info os.FileInfo, err error) error {
		var data []byte
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != UserExt {
			return nil
		}
		var user UserSettings
		data, err = ioutil.ReadFile(path)
		if err = yaml.Unmarshal(data, &user); err != nil {
			return err
		}
		if _, ok := storage.Users[user.ID]; ok {
			userSettings[user.ID] = user
		}
		return err
	})
	return err
}

func NewUser(nickname string) error {
	user := User{
		Nickname: nickname,
	}
	if !lib.ValidateSysName(nickname) {
		return fmt.Errorf(Lang(DefLang, `invalidfield`), Lang(DefLang, `nickname`))
	}
	for _, item := range storage.Users {
		if item.Nickname == nickname {
			return fmt.Errorf(Lang(DefLang, `errnickname`), nickname)
		}

	}
	private, public, err := lib.GenerateKeys()
	if err != nil {
		return err
	}
	user.PublicKey = public
	user.ID = crc32.ChecksumIEEE(private)
	if err = ioutil.WriteFile(filepath.Join(cfg.Users.Dir, user.Nickname+`.key`),
		[]byte(hex.EncodeToString(private)), 0777 /*os.ModePerm*/); err != nil {
		return err
	}
	storage.Users[user.ID] = &user
	userSettings[user.ID] = UserSettings{
		ID:   user.ID,
		Lang: appInfo.Lang,
	}
	return nil
}

// AddHistoryEditor adds the history item to the user's settings
func AddHistoryEditor(id uint32, name string) error {
	var (
		cur UserSettings
	)
	cur = userSettings[id]
	ret := make([]string, 1, HistoryLimit+1)
	ret[0] = name
	for _, item := range cur.History.Editor {
		if item != name {
			ret = append(ret, item)
			if len(ret) == HistoryLimit {
				break
			}
		}
	}
	cur.History.Editor = ret
	userSettings[id] = cur
	return SaveUser(id)
}

// GetHistory returns the history list
func GetHistory(c echo.Context, list []string) []ScriptItem {
	ret := make([]ScriptItem, 0, len(list))
	for _, item := range list {
		script := getScript(item)
		if script == nil {
			continue
		}
		ret = append(ret, ScriptItem{
			Name: item,
			Title: es.ReplaceVars(script.Settings.Title, script.Langs[c.(*Auth).Lang],
				&langRes[GetLangId(c.(*Auth).User)]),
		})
	}
	return ret
}

// GetHistoryEditor returns the history list
func GetHistoryEditor(c echo.Context) []ScriptItem {
	return GetHistory(c, userSettings[c.(*Auth).User.ID].History.Editor)
}

// LatestHistory returns the latest open project
func LatestHistoryEditor(c echo.Context) (ret string) {
	list := GetHistoryEditor(c)
	if len(list) > 0 {
		return list[0].Name
	}
	return
}

// AddHistoryRun adds the launched item to the user's settings
func AddHistoryRun(id uint32, name string) error {
	var (
		cur UserSettings
	)
	cur = userSettings[id]
	ret := make([]string, 1, RunLimit+1)
	ret[0] = name
	for _, item := range cur.History.Run {
		if item != name {
			ret = append(ret, item)
			if len(ret) == RunLimit {
				break
			}
		}
	}
	cur.History.Run = ret
	userSettings[id] = cur
	return SaveUser(id)
}

// GetHistoryRun returns the launchedhistory list
/*func GetHistoryRun(id uint32) []ScriptItem {
	return GetHistory(userSettings[id].History.Run)
}*/

func SaveUser(id uint32) error {
	data, err := yaml.Marshal(userSettings[id])
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(cfg.Users.Dir,
		storage.Users[id].Nickname+UserExt), data, 0777 /*os.ModePerm*/)
}
