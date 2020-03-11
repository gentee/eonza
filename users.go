// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"eonza/lib"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	HistEditor = iota
)

type History struct {
	Editor []string `yaml:"editor"`
}

// UserSettings stores the user's settings
type UserSettings struct {
	ID      uint32  `yaml:"id"`
	Lang    string  `yaml:"lang"`
	History History `yaml:"history"`

	index int // index in cfg.Users
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
	for i, item := range storage.Users {
		userSettings[item.ID] = UserSettings{
			ID:    item.ID,
			Lang:  appInfo.Lang,
			index: i,
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
		if curUser, ok := userSettings[user.ID]; ok {
			user.index = curUser.index
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
		return fmt.Errorf(Lang(`invalidfield`), Lang(`nickname`))
	}
	mutex.Lock()
	defer mutex.Unlock()
	for _, item := range storage.Users {
		if item.Nickname == nickname {
			return fmt.Errorf(Lang(`errnickname`), nickname)
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
	storage.Users = append(storage.Users, user)
	return nil
}

// AddHistory adds the history item to the user's settings
func AddHistory(id uint32, history int, name string) error {
	mutex.Lock()
	var (
		cur UserSettings
		in  []string
	)
	cur = userSettings[id]
	if history == HistEditor {
		in = cur.History.Editor
	}
	ret := make([]string, 1, HistoryLimit+1)
	ret[0] = name
	for _, item := range in {
		if item != name {
			ret = append(ret, item)
			if len(ret) == HistoryLimit {
				break
			}
		}
	}
	if history == HistEditor {
		cur.History.Editor = ret
		userSettings[id] = cur
	}
	mutex.Unlock()
	return SaveUser(id)
}

// GetHistory returns the history list
func GetHistory(id uint32, history int) []string {
	mutex.RLock()
	defer mutex.RUnlock()
	if history == HistEditor {
		return userSettings[id].History.Editor
	}
	return userSettings[id].History.Editor
}

// LatestHistory returns the latest open project
func LatestHistory(id uint32, history int) (ret string) {
	list := GetHistory(id, history)
	if len(list) > 0 {
		ret = list[0]
	}
	return
}

func SaveUser(id uint32) error {
	mutex.RLock()
	defer mutex.RUnlock()

	data, err := yaml.Marshal(userSettings[id])
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(cfg.Users.Dir,
		storage.Users[userSettings[id].index].Nickname+UserExt), data, 0777 /*os.ModePerm*/)
}
