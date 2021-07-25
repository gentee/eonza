// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"eonza/lib"
	es "eonza/script"
	"eonza/users"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	ChromeExtension = ``
)

type BrowserSettings struct {
	HTML bool `json:"html"`
}

type Browser struct {
	ID       uint32          `json:"id"`
	URLs     string          `json:"urls"`
	Scripts  string          `json:"scripts"`
	Settings BrowserSettings `json:"settings"`
}

type BrowsersResponse struct {
	List  []*Browser `json:"list"`
	Error string     `json:"error,omitempty"`
}

type ExtScript struct {
	Name     string          `json:"name"`
	Title    string          `json:"title"`
	Settings BrowserSettings `json:"settings"`
}

type ExtRun struct {
	Name  string `json:"name"`
	Open  bool   `json:"open"`
	URL   string `json:"url"`
	Title string `json:"title"`
	HTML  string `json:"html,omitempty"`
}

type ExtListResponse struct {
	List  []ExtScript `json:"list,omitempty"`
	Error string      `json:"error,omitempty"`
}

type ExtInfo struct {
	Url string `json:"url"`
}

func browserExtHandle(c echo.Context) error {
	var (
		err error
		ext ExtInfo
	)
	if err = c.Bind(&ext); err != nil {
		return jsonError(c, err)
	}
	lang := c.(*Auth).Lang
	glob := &langRes[GetLangId(c.(*Auth).User)]
	list := make([]ExtScript, 0)
	added := make(map[string]bool)
	for _, item := range storage.Browsers {
		url := strings.ReplaceAll(strings.TrimSpace(item.URLs), "\n", " ")
		match := len(url) == 0
		if !match {
			for _, upath := range strings.Split(url, " ") {
				upath = strings.TrimSpace(upath)
				if strings.HasPrefix(upath, `http`) {
					match = strings.HasPrefix(ext.Url, upath)
				} else {
					match = strings.Contains(ext.Url, upath)
				}
				if match {
					break
				}
			}
		}
		if match {
			for _, cmd := range strings.Split(item.Scripts, `,`) {
				cmd = strings.TrimSpace(cmd)
				if added[cmd] {
					continue
				}
				var script *Script
				if script = getScript(cmd); script == nil {
					continue
				}
				user := c.(*Auth).User
				if ScriptAccess(script.Settings.Name, script.Settings.Path, user.RoleID) == nil {
					list = append(list, ExtScript{
						Name:     script.Settings.Name,
						Title:    es.ReplaceVars(script.Settings.Title, script.Langs[lang], glob),
						Settings: item.Settings,
					})
					added[cmd] = true
				}
			}
		}
	}
	return c.JSON(http.StatusOK, &ExtListResponse{
		List: list,
	})
}

func browserRunHandle(c echo.Context) error {
	var (
		err  error
		ext  ExtRun
		data []byte
	)
	if err = c.Bind(&ext); err != nil {
		return jsonError(c, err)
	}
	if data, err = json.Marshal(ext); err != nil {
		return jsonError(c, err)
	}
	user := c.(*Auth).User
	rs := RunScript{
		Name: ext.Name,
		Open: ext.Open && cfg.HTTP.Host == Localhost,
		Data: string(data),
		User: *user,
		/*		users.User{
				ID:       users.XAdminID,
				Nickname: users.RootUser,
				RoleID:   users.BrowserID,
			},*/
		Role: users.Role{
			ID:   users.BrowserID,
			Name: users.BrowserRole,
		},
		IP: Localhost,
	}
	if err = systemRun(&rs); err != nil {
		NewNotification(&Notification{
			Text:   fmt.Sprintf(`Browser extension error: %s`, err.Error()),
			UserID: user.ID,
			RoleID: users.BrowserID,
			Script: rs.Name,
		})
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, &RunResponse{Success: true, Port: rs.Port, ID: rs.ID})
}

func browsersResponse(c echo.Context) error {
	return c.JSON(http.StatusOK, &BrowsersResponse{
		List: storage.Browsers,
	})
}

func browsersHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	return browsersResponse(c)
}

func saveBrowserHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	var browser Browser
	if err := c.Bind(&browser); err != nil {
		return jsonError(c, err)
	}
	if len(browser.Scripts) == 0 {
		return jsonError(c, Lang(DefLang, `errreq`, `Scripts`))
	}
	isBrowser := func(id uint32) int {
		for i, item := range storage.Browsers {
			if item.ID == id {
				return i + 1
			}
		}
		return 0
	}
	curID := isBrowser(browser.ID)
	if browser.ID == 0 {
		for {
			browser.ID = lib.RndNum()
			if isBrowser(browser.ID) == 0 {
				break
			}
		}
		storage.Browsers = append(storage.Browsers, &browser)
	} else if curID == 0 {
		return jsonError(c, fmt.Errorf(`Access denied`))
	} else {
		storage.Browsers[curID-1] = &browser
	}
	if err := SaveStorage(); err != nil {
		return jsonError(c, err)
	}
	return browsersResponse(c)
}

func removeBrowserHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}

	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	for i, item := range storage.Browsers {
		if item.ID == uint32(id) {
			if i < len(storage.Browsers)-1 {
				storage.Browsers = append(storage.Browsers[:i], storage.Browsers[i+1:]...)
			} else {
				storage.Browsers = storage.Browsers[:i]
			}
			if err := SaveStorage(); err != nil {
				return jsonError(c, err)
			}
			break
		}
	}
	return browsersResponse(c)
}
