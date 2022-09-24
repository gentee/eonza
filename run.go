// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"eonza/script"
	"eonza/users"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type RunScript struct {
	Name    string
	Open    bool
	Console bool
	User    users.User
	Role    users.Role
	IP      string
	Data    string

	// Result fields
	ID      uint32
	Port    int
	Encoded []byte
}

func systemRun(rs *RunScript) error {
	var (
		item     *Script
		src      string
		langCode string
		langid   int
	)
	port, err := getPort()
	if err != nil {
		return err
	}
	localPort, err := getPort()
	if err != nil {
		return err
	}
	var (
		formAlign uint32
		userID    uint32
	)
	if rs.Role.ID >= users.ResRoleID {
		utemp, _ := GetUser(users.XRootID)
		langCode = GetLangCode(&utemp)
		langid = GetLangId(&utemp)
		userID = utemp.ID
	} else {
		langCode = GetLangCode(&rs.User)
		langid = GetLangId(&rs.User)
		userID = rs.User.ID
	}
	if u, ok := userSettings[userID]; ok {
		formAlign = u.FormAlign
	}
	if item = getRunScript(rs.Name); item == nil {
		return fmt.Errorf(Lang(langid, `erropen`, rs.Name))
	}
	if err = ScriptAccess(item.Settings.Name, item.Settings.Path, rs.Role.ID); err != nil {
		return err
	}
	if item.Settings.Unrun {
		return fmt.Errorf(Lang(langid, `errnorun`, rs.Name))
	}
	title := item.Settings.Title
	if langTitle := strings.Trim(title, `#`); langTitle != title {
		if val, ok := item.Langs[langCode][langTitle]; ok {
			title = val
		} else if val, ok := item.Langs[LangDefCode][langTitle]; ok {
			title = val
		}
	}
	var cdn string
	if !lib.IsPrivateHost(cfg.HTTP.Host) {
		cdn = fmt.Sprintf(`https://%s:%d`, cfg.HTTP.Host, cfg.HTTP.Port)
	}
	header := script.Header{
		Name:         rs.Name,
		Title:        title,
		PackagesDir:  cfg.PackagesDir,
		AssetsDir:    cfg.AssetsDir,
		LogDir:       cfg.Log.Dir,
		CDN:          cdn,
		Data:         rs.Data,
		Console:      rs.Console,
		IsPlayground: cfg.playground,
		IsAutoFill:   IsAutoFill(),
		IP:           rs.IP,
		User:         rs.User,
		Role:         rs.Role,
		ClaimKey:     cfg.HTTP.JWTKey + sessionKey,
		IsPro:        IsProActive(), //storage.Trial.Mode > TrialOff,
		Constants:    storage.Settings.Constants,
		SecureConsts: SecureConstants(),
		Lang:         langCode,
		TaskID:       lib.RndNum(),
		FormAlign:    formAlign,
		ServerPort:   cfg.HTTP.LocalPort,
		URLPort:      cfg.HTTP.Port,
		HTTP: &lib.HTTPConfig{
			Host:      cfg.HTTP.Host,
			Port:      port,
			LocalPort: localPort,
			Open:      rs.Open,
			Theme:     cfg.HTTP.Theme,
			Cert:      cfg.HTTP.Cert,
			Priv:      cfg.HTTP.Priv,
		},
	}
	if len(item.pkg) > 0 {
		header.PkgPath = filepath.Join(cfg.PackagesDir, item.pkg)
	}
	if header.IsPlayground {
		header.Playground = &cfg.Playground
		tasksLimit := cfg.Playground.Tasks
		for _, item := range tasks {
			if item.Status < TaskFinished {
				tasksLimit--
			}
		}
		if tasksLimit <= 0 {
			return fmt.Errorf(Lang(langid, `errtasklimit`, cfg.Playground.Tasks))
		}
	}
	if src, err = GenSource(item, &header); err != nil {
		return err
	}
	if storage.Settings.IncludeSrc {
		if header.SourceCode, err = lib.GzipCompress([]byte(src)); err != nil {
			return err
		}
	}
	data, err := script.Encode(header, src)
	if err != nil {
		return err
	}
	if !Licensed() && storage.Trial.Mode == TrialOn {
		now := time.Now()
		if storage.Trial.Last.Day() != now.Day() {
			storage.Trial.Count++
			storage.Trial.Last = now
			if storage.Trial.Count > TrialDays {
				storage.Trial.Mode = TrialDisabled
				SetActive()
			}
			if err = SaveStorage(); err != nil {
				return err
			}
		}
	}
	if err = NewTask(header); err != nil {
		return err
	}
	if rs.Console {
		rs.Encoded = data.Bytes()
	}
	rs.Port = header.HTTP.Port
	rs.ID = header.TaskID

	return nil
}
