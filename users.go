// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"path/filepath"

	es "eonza/script"
	"eonza/users"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v2"
)

type Fav struct {
	Name     string `json:"name" yaml:"name"`
	IsFolder bool   `json:"isfolder" yaml:"isfolder,omitempty"`
	Children []Fav  `json:"children,omitempty" yaml:"children,omitempty"`
}

type History struct {
	Editor []string `yaml:"editor"`
	Run    []string `yaml:"run"`
}

// UserSettings stores the user's settings
type UserSettings struct {
	ID      uint32  `json:"id" yaml:"id"`
	Lang    string  `json:"lang" yaml:"lang"`
	History History `json:"history" yaml:"history"`
	Favs    []Fav   `json:"favs" yaml:"favs"`
}

// User stores user's parameters
type User struct { // Deprecated
	ID        uint32
	Nickname  string
	PublicKey []byte
}

var (
	userSettings = make(map[uint32]UserSettings)
)

func LoadUsersSettings() error {
	var err error
	for _, item := range users.Users {
		var (
			data []byte
			user UserSettings
		)
		user.Lang = appInfo.Lang
		data, err = ioutil.ReadFile(filepath.Join(cfg.Users.Dir, item.Nickname+UserExt))
		if err == nil {
			if err = yaml.Unmarshal(data, &user); err != nil {
				return err
			}
		}
		user.ID = item.ID
		userSettings[user.ID] = user
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
		users.Users[id].Nickname+UserExt), data, 0777 /*os.ModePerm*/)
}

func RootUserSettings() UserSettings {
	return userSettings[users.RootID]
}
