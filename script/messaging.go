// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"encoding/json"
	"eonza/lib"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
	"github.com/kataras/golog"
	mail "github.com/xhit/go-simple-mail/v2"
)

type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

var ErrInvalidPar = fmt.Errorf(`Invalid parameter`)

func ObjStr(obj *core.Obj, key string) string {
	val, err := vm.ItemºObjStr(obj, key)
	if err != nil || val == nil || val.Data == nil {
		return ``
	}
	return fmt.Sprint(val.Data)
}

func ObjInt(obj *core.Obj, key string) int64 {
	val := ObjStr(obj, key)
	ret, err := strconv.ParseInt(val, 0, 64)
	if err != nil {
		return 0
	}
	return ret
}

func SendEmail(smtpserv *core.Obj, emailobj *core.Obj) error {
	if vm.Type(smtpserv) != `map.obj` || vm.Type(emailobj) != `map.obj` {
		return ErrInvalidPar
	}
	server := mail.NewSMTPClient()
	server.Host = ObjStr(smtpserv, "host")
	if len(server.Host) == 0 {
		server.Host = `localhost`
	}
	server.Port = int(ObjInt(smtpserv, "port"))
	server.Username = ObjStr(smtpserv, "username")
	server.Password = ObjStr(smtpserv, "password")
	if ObjStr(smtpserv, "consec") == `ssl` {
		server.Encryption = mail.EncryptionSSL
		if server.Port == 0 {
			server.Port = 465
		} else if server.Port == 587 {
			server.Encryption = mail.EncryptionTLS
		}
	} else {
		if server.Port == 0 {
			server.Port = 25
		}
		server.Encryption = mail.EncryptionNone
	}
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	client, err := server.Connect()
	if err != nil {
		return err
	}
	email := mail.NewMSG()
	from := ObjStr(emailobj, "from")
	if len(from) == 0 {
		from = server.Username
	}
	to := ObjStr(emailobj, "to")
	subject := ObjStr(emailobj, "subject")
	body := ObjStr(emailobj, "body")
	if len(to) == 0 || len(subject) == 0 || len(body) == 0 {
		return ErrInvalidPar
	}
	email.SetFrom(from).AddTo(to).SetSubject(subject)
	method := mail.TextPlain
	if strings.HasPrefix(body, `<html`) {
		method = mail.TextHTML
	}
	email.SetBody(method, body)
	//	email.SetPriority(mail.PriorityHigh)
	err = email.Send(client)
	return err
}

func SendNotification(msg string) (err error) {
	if _, err = lib.LocalPost(scriptTask.Header.ServerPort, `api/notification`, PostNfy{
		TaskID: scriptTask.Header.TaskID,
		Text:   msg,
		Script: scriptTask.Header.Name,
	}); err != nil {
		golog.Error(err)
	}
	return
}

func RunScript(script, data string, silent int64) (err error) {
	var body []byte
	if body, err = lib.LocalPost(scriptTask.Header.ServerPort, `api/runscript`, PostScript{
		TaskID: scriptTask.Header.TaskID,
		Script: script,
		Data:   data,
		UserID: scriptTask.Header.User.ID,
		RoleID: scriptTask.Header.Role.ID,
		Silent: silent != 0,
	}); err == nil {
		var answer Response
		if err = json.Unmarshal(body, &answer); err == nil {
			if !answer.Success && len(answer.Error) > 0 {
				err = fmt.Errorf(answer.Error)
			}
		}
	}
	return
}
