// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"eonza/lib"
	"eonza/users"
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	es "eonza/script"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

const (
	NfyExt       = `eon`
	NfyPageLimit = 25
	NfyLimit     = 50 // save
)

type NfyResponse struct {
	Unread int    `json:"unread"`
	List   []Nfy  `json:"list,omitempty"`
	Error  string `json:"error,omitempty"`
}

type LatestResponse struct {
	Version     string `json:"version"`
	Notify      string `json:"notify"`
	LastChecked string `json:"lastchecked"`
	Error       string `json:"error,omitempty"`
}

type Nfy struct {
	Text   string `json:"text"`
	Time   string `json:"time"`
	Hash   string `json:"hash"`
	ToDel  bool   `json:"todel"`
	Script string `json:"script"`
	User   string `json:"user"`
	Role   string `json:"role"`
}

type Notification struct {
	Hash   uint64
	Text   string
	Time   time.Time
	UserID uint32
	RoleID uint32
	Script string
}

type VerUpdate struct {
	Version     string   `json:"version"`
	Langs       []string `json:"langs"`
	Changelog   string   `json:"changelog"`
	Downloads   string   `json:"downloads"`
	Notify      string   `json:"notify,omitempty"`
	LastChecked time.Time
}

type Notifications struct {
	Unread   int // deprecated
	ReadTime map[uint32]time.Time
	List     []*Notification
	Update   VerUpdate
}

var (
	nfyData  = Notifications{ReadTime: make(map[uint32]time.Time)}
	nfyMutex = &sync.Mutex{}
	CRCTable = crc64.MakeTable(crc64.ISO)
)

func LoadNotifications() {
	nfyfile := lib.ChangeExt(cfg.path, NfyExt)
	if _, err := os.Stat(nfyfile); err != nil {
		if os.IsNotExist(err) {
			return
		}
		golog.Fatal(err)
	}
	data, err := ioutil.ReadFile(nfyfile)
	if err != nil {
		golog.Fatal(err)
	}
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	if err = dec.Decode(&nfyData); err != nil {
		golog.Fatal(err)
	}
}

func NewNotification(nfy *Notification) (err error) {
	if len(nfy.Text) == 0 {
		return
	}
	nfyMutex.Lock()
	defer nfyMutex.Unlock()
	nfy.Time = time.Now()
	nfy.Hash = crc64.Checksum([]byte(fmt.Sprintf(`%s%d`, nfy.Text, nfy.UserID)), CRCTable)
	shift := 0
	if len(nfyData.List) >= NfyLimit {
		shift = 1
	}
	for i, item := range nfyData.List {
		if item.Hash == nfy.Hash {
			shift++
			continue
		}
		if shift > 0 && i >= shift {
			nfyData.List[i-shift] = item
		}
	}
	if shift > 0 {
		off := len(nfyData.List) - shift
		nfyData.List[off] = nfy
		if shift > 1 {
			for i := 1; i < shift; i++ {
				nfyData.List[off+i] = nil
			}
			nfyData.List = nfyData.List[:off+1]
		}
	} else {
		nfyData.List = append(nfyData.List, nfy)
	}
	return saveNotifications(true)
}

func saveNotifications(update bool) error {
	var (
		data bytes.Buffer
		err  error
	)
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(nfyData); err != nil {
		return err
	}
	if err = ioutil.WriteFile(lib.ChangeExt(cfg.path, NfyExt), data.Bytes(),
		0777 /*os.ModePerm*/); err != nil {
		return err
	}
	if !update {
		return nil
	}
	var out []byte
	for id, client := range clients {
		resp := NfyList(false, client.UserID, client.RoleID)
		if out, err = json.Marshal(resp); err == nil {
			cmd := WsCmd{
				//		    TaskID:   postNfy.TaskID,
				Cmd:     WcNotify,
				Message: string(out),
			}
			err := client.Conn.WriteJSON(cmd)
			if err != nil {
				client.Conn.Close()
				delete(clients, id)
			}
		}
	}
	return err
}

func NfyList(clear bool, userid, roleid uint32) *NfyResponse {
	var (
		nfyFlag int
	)
	if roleid != users.XAdminID {
		if role, ok := GetRole(roleid); ok {
			nfyFlag = role.Notifications
		}
	}

	nlen := len(nfyData.List)
	slen := nlen
	if slen > NfyPageLimit {
		slen = NfyPageLimit
	}
	ret := make([]Nfy, 0, slen)
	var unread int
	for i := 0; i < slen; i++ {
		item := nfyData.List[nlen-i-1]
		if roleid == users.XAdminID || (nfyFlag&4 == 4) ||
			(nfyFlag&1 == 1 && userid == item.UserID) ||
			(nfyFlag&2 == 2 && roleid == item.RoleID) {
			todel := roleid == users.XAdminID || (nfyFlag&0x400 == 0x400) ||
				(nfyFlag&0x100 == 0x100 && userid == item.UserID) ||
				(nfyFlag&0x200 == 0x200 && roleid == item.RoleID)
			var userName, roleName string

			if item.UserID != users.XRootID {
				userName, roleName = GetUserRole(item.UserID, item.RoleID)
			}
			ret = append(ret, Nfy{
				Hash:   strconv.FormatUint(item.Hash, 10),
				Text:   strings.ReplaceAll(item.Text, "\n", "<br>"),
				Time:   item.Time.Format(TimeFormat),
				ToDel:  todel,
				Script: item.Script,
				User:   userName,
				Role:   roleName,
			})
			if item.Time.After(nfyData.ReadTime[userid]) {
				unread++
			}
		}
	}
	resp := NfyResponse{
		List:   ret,
		Unread: unread,
	}
	if clear {
		nfyData.ReadTime[userid] = time.Now()
	}
	return &resp
}

func nfyHandle(c echo.Context) error {
	nfyMutex.Lock()
	defer nfyMutex.Unlock()
	user := c.(*Auth).User
	resp := NfyList(true, user.ID, user.RoleID)
	if resp.Unread != 0 {
		saveNotifications(false)
	}
	return c.JSON(http.StatusOK, resp)
}

func notificationHandle(c echo.Context) error {
	var (
		postNfy es.PostNfy
		err     error
	)
	if !strings.HasPrefix(c.Request().Host, Localhost+`:`) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}
	if err = c.Bind(&postNfy); err != nil {
		return jsonError(c, err)
	}
	nfy := Notification{
		Text:   postNfy.Text,
		UserID: users.XRootID,
		RoleID: users.XAdminID,
		Script: postNfy.Script,
	}
	if ptask, ok := tasks[postNfy.TaskID]; ok {
		nfy.UserID = ptask.UserID
		nfy.RoleID = ptask.RoleID
	}
	if err = NewNotification(&nfy); err != nil {
		return jsonError(c, err)
	}
	return jsonSuccess(c)
}

func removeNfyHandle(c echo.Context) error {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	isAccess := func(item *Notification) bool {
		user := c.(*Auth).User
		if user.RoleID != users.XAdminID {
			var access bool
			if role, ok := GetRole(user.RoleID); ok {
				nfyFlag := role.Notifications
				access = (nfyFlag&0x400 == 0x400) ||
					(nfyFlag&0x100 == 0x100 && user.ID == item.UserID) ||
					(nfyFlag&0x200 == 0x200 && user.RoleID == item.RoleID)
			}
			if !access {
				return false
			}
		}
		return true
	}

	nfyMutex.Lock()
	defer nfyMutex.Unlock()
	shift := 0
	for i, item := range nfyData.List {
		if item.Hash == id {
			shift++
			if !isAccess(item) {
				return jsonError(c, fmt.Errorf(`Access denied`))
			}
			continue
		}
		if shift > 0 {
			nfyData.List[i-shift] = item
		}
	}
	if shift > 0 {
		off := len(nfyData.List) - shift
		for i := 0; i < shift; i++ {
			nfyData.List[off+i] = nil
		}
		nfyData.List = nfyData.List[:off]
	}
	if err := saveNotifications(true); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}

func GetNewVersion(lang string) (ret string) {
	if len(nfyData.Update.Version) > 0 {
		var (
			lid  int
			pref string
		)
		for _, item := range nfyData.Update.Langs {
			if item == lang {
				lid = langsId[lang]
				pref = lang + `/`
				break
			}
		}
		ret = fmt.Sprintf(`%s: <span style="padding: 4px 8px;
	font-weight: bold;background-color: #ffff00">%s</span><br><a style="margin-right: 2rem;" href="%s" target="_blank">%s</a><a href="%s" target="_blank">%s</a>`, Lang(lid, `newver`),
			nfyData.Update.Version, appInfo.Homepage+pref+nfyData.Update.Changelog,
			Lang(lid, `changelog`),
			appInfo.Homepage+pref+nfyData.Update.Downloads, Lang(lid, `downloads`))
	}
	return
}

func CheckUpdates() error {
	resp, err := http.Get(appInfo.Homepage + `latest`)
	if err != nil {
		return err
	}
	if body, err := ioutil.ReadAll(resp.Body); err == nil {
		var upd VerUpdate
		if err = json.Unmarshal(body, &upd); err == nil {
			if len(upd.Version) > 0 && upd.Version != Version {
				nfyData.Update.Version = upd.Version
				nfyData.Update.Changelog = upd.Changelog
				nfyData.Update.Downloads = upd.Downloads
				nfyData.Update.Langs = upd.Langs
			}
		}
		resp.Body.Close()
	}
	nfyMutex.Lock()
	defer nfyMutex.Unlock()
	nfyData.Update.LastChecked = time.Now()
	return saveNotifications(true)
}

func AutoCheckUpdate() {
	var (
		update bool
	)
	now := time.Now()
	switch storage.Settings.AutoUpdate {
	case `daily`:
		update = now.After(nfyData.Update.LastChecked.Add(time.Hour * 24))
	case `weekly`:
		update = now.After(nfyData.Update.LastChecked.Add(time.Hour * 24 * 7))
	case `mothly`:
		update = now.After(nfyData.Update.LastChecked.Add(time.Hour * 24 * 30))
	}
	if !update {
		return
	}
	if err := CheckUpdates(); err != nil {
		return
	}
	if nfy := GetNewVersion(RootUserSettings().Lang); len(nfy) > 0 {
		NewNotification(&Notification{Text: nfy, UserID: users.XRootID, RoleID: users.XAdminID})
	}
	return
}

func latestVerHandle(c echo.Context) error {
	if err := CheckUpdates(); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, LatestResponse{
		Version:     nfyData.Update.Version,
		Notify:      GetNewVersion(GetLangCode(c.(*Auth).User)),
		LastChecked: nfyData.Update.LastChecked.Format(TimeFormat),
	})
}
