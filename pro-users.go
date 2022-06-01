// Copyright 2021 Alexey Krivonogov. All rights reserved.

package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kataras/golog"
)

type UserPro struct {
	Forms map[string][]map[string]interface{}
	path  string
}

var (
	usersPro  = make(map[uint32]*UserPro)
	usersPath string
)

func LoadUsers(path string) {
	usersPath = path
	for _, u := range proStorage.Users {
		var user UserPro

		path := filepath.Join(path, fmt.Sprintf("%x.pro", u.ID))
		if _, err := os.Stat(path); err == nil {
			if data, err := os.ReadFile(path); err == nil {
				user.path = path
				dec := gob.NewDecoder(bytes.NewBuffer(data))
				if err = dec.Decode(&user); err != nil {
					golog.Error(err)
				}
			}
		}
		usersPro[u.ID] = &user
	}
}

func DeleteUser(id uint32) {
	if u, ok := usersPro[id]; ok {
		if len(u.path) > 0 {
			os.Remove(u.path)
		}
		delete(usersPro, id)
	}
}

func SetUserForms(id uint32, ref string, m map[string]interface{}) error {
	var (
		user *UserPro
		ok   bool
	)

	if user, ok = usersPro[id]; !ok {
		var u UserPro
		user = &u
		usersPro[id] = user
	}
	if user.Forms == nil {
		user.Forms = make(map[string][]map[string]interface{})
	}

	if _, ok := m[`_name`]; !ok {
		m[`_name`] = ``
	}
	m[`_time`] = time.Now().Unix()
	if !m["_afcheck"].(bool) {
		m[`_name`] = `---`
	}
	name := m[`_name`]
	ret := []map[string]interface{}{m}

	for _, v := range user.Forms[ref] {
		if fmt.Sprint(v["_name"]) == name || !v["_afcheck"].(bool) {
			continue
		}
		if t, ok := v["_time"]; ok {
			prev := time.Unix(t.(int64), 0)
			if time.Now().After(prev.AddDate(0, 0, 50)) {
				continue
			}
		} else {
			continue
		}
		ret = append(ret, v)
		if len(ret) == 7 {
			break
		}
	}
	user.Forms[ref] = ret
	return SaveUserSettings(id)
}

func SaveUserSettings(id uint32) error {
	if u, ok := usersPro[id]; ok {
		if len(u.path) == 0 {
			u.path = filepath.Join(usersPath, fmt.Sprintf("%x.pro", id))
		}
		var (
			data bytes.Buffer
			err  error
		)
		enc := gob.NewEncoder(&data)
		if err = enc.Encode(u); err != nil {
			return err
		}
		if err = os.WriteFile(u.path, data.Bytes(), 0777 /*os.ModePerm*/); err != nil {
			return err
		}
	}
	return nil
}
