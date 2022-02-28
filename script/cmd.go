// Copyright 2022 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
)

type CmdData struct {
	TaskID   uint32
	Unique   uint32
	Finished bool
	Cmd      string
	Error    string
	Value    interface{}
}

const (
	CmdStart    = "start"
	CmdPing     = "ping"
	CmdShutdown = "shutdown"
)

func SendCmd(port int, cmd *CmdData) (ret *CmdData, err error) {
	var (
		resp   *http.Response
		data   bytes.Buffer
		answer CmdData
	)
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(cmd); err != nil {
		return
	}
	resp, err = http.Post(fmt.Sprintf("http://localhost:%d/cmd", port),
		"application/octet-stream", &data)
	if err == nil {
		dec := gob.NewDecoder(resp.Body)
		ret = &answer
		if err = dec.Decode(ret); err == nil {
			if ret.TaskID != cmd.TaskID {
				err = fmt.Errorf(`wrong task %d != %d in SendCmd`, cmd.TaskID, ret.TaskID)
			} else if len(ret.Error) > 0 {
				err = fmt.Errorf(ret.Error)
			}
		}
		resp.Body.Close()
	}
	return
}

func ProcessCmd(taskid uint32, r io.Reader) (ret *CmdData, err error) {
	var answer CmdData

	dec := gob.NewDecoder(r)
	ret = &answer
	if err = dec.Decode(ret); err == nil {
		if ret.TaskID != taskid {
			err = fmt.Errorf(`wrong task %d != %d in ProcessCmd`, taskid, ret.TaskID)
		} else if len(ret.Error) > 0 {
			err = fmt.Errorf(ret.Error)
		}
	}
	return
}

func ResponseCmd(cmd *CmdData) []byte {
	var data bytes.Buffer

	enc := gob.NewEncoder(&data)
	if err := enc.Encode(cmd); err != nil {
		enc.Encode(CmdData{
			TaskID: cmd.TaskID,
			Error:  err.Error(),
		})
	}
	return data.Bytes()
}
