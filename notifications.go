// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"eonza/lib"
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	es "eonza/script"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

const (
	NfyExt   = `nfy`
	NfyLimit = 25
)

type NfyResponse struct {
	Unread int    `json:"unread"`
	List   []Nfy  `json:"list,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Nfy struct {
	Text string `json:"text"`
	Time string `json:"time"`
	Hash string `json:"hash"`
}

type Notification struct {
	Hash uint64
	Text string
	Time time.Time
}

type Notifications struct {
	Unread int
	List   []*Notification
}

var (
	nfyData  = Notifications{Unread: 0, List: make([]*Notification, 0)}
	nfyHash  = make(map[uint64]int)
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
	for i, item := range nfyData.List {
		nfyHash[item.Hash] = i
	}
}

func NewNotification(nfy *Notification) (err error) {
	if len(nfy.Text) == 0 {
		return
	}
	nfyMutex.Lock()
	defer nfyMutex.Unlock()
	nfy.Time = time.Now()
	nfy.Hash = crc64.Checksum([]byte(nfy.Text), CRCTable)
	fmt.Println(`new`, nfy.Text)
	if i, ok := nfyHash[nfy.Hash]; ok {
		nlen := len(nfyData.List) - 1
		if i < nlen {
			copy(nfyData.List[i:], nfyData.List[i+1:])
		}
		nfyData.List[nlen] = nil
		nfyData.List = nfyData.List[:nlen]
		if i <= nlen-nfyData.Unread {
			nfyData.Unread++
		}
	} else {
		nfyData.Unread++
	}
	nfyHash[nfy.Hash] = len(nfyData.List)
	nfyData.List = append(nfyData.List, nfy)
	fmt.Println(`All`, nfyData.List, nfyData.List[0])
	if err = saveNotifications(); err == nil {
		var out []byte
		if out, err = json.Marshal(NfyList(false)); err == nil {
			cmd := WsCmd{
				//		    TaskID:   postNfy.TaskID,
				Cmd:     WcNotify,
				Message: string(out),
			}
			for id, client := range clients {
				err := client.Conn.WriteJSON(cmd)
				if err != nil {
					client.Conn.Close()
					delete(clients, id)
				}
			}
		}
	}
	return err
}

func saveNotifications() error {
	var (
		data bytes.Buffer
		err  error
		infy Notifications
	)
	enc := gob.NewEncoder(&data)
	infy.Unread = nfyData.Unread
	if len(nfyData.List) > 4*NfyLimit {
		infy.List = nfyData.List[len(nfyData.List)-4*NfyLimit:]
	} else {
		infy.List = nfyData.List
	}
	if err = enc.Encode(infy); err != nil {
		return err
	}
	return ioutil.WriteFile(lib.ChangeExt(cfg.path, NfyExt), data.Bytes(), 0777 /*os.ModePerm*/)
}

func NfyList(clear bool) *NfyResponse {
	nlen := len(nfyData.List)
	slen := nlen
	if slen > NfyLimit {
		slen = NfyLimit
	}
	ret := make([]Nfy, slen)
	for i := 0; i < slen; i++ {
		item := nfyData.List[nlen-i-1]
		ret[i] = Nfy{
			Hash: strconv.FormatUint(item.Hash, 10),
			Text: item.Text,
			Time: item.Time.Format(TimeFormat),
		}
	}
	resp := NfyResponse{
		List:   ret,
		Unread: nfyData.Unread,
	}
	if clear {
		nfyData.Unread = 0
	}
	return &resp
}

func nfyHandle(c echo.Context) error {
	nfyMutex.Lock()
	defer nfyMutex.Unlock()
	return c.JSON(http.StatusOK, NfyList(true))
}

func notificationHandle(c echo.Context) error {
	var (
		postNfy es.PostNfy
		err     error
	)
	if err = c.Bind(&postNfy); err != nil {
		return jsonError(c, err)
	}
	if err = NewNotification(&Notification{
		Text: postNfy.Text,
	}); err != nil {
		return jsonError(c, err)
	}
	return jsonSuccess(c)
}
