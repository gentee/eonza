// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"encoding/json"
	"eonza/lib"
	"strings"
	"time"
)

type ExtData struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	HTML  string `json:"html,omitempty"`
}

type ExtFill struct {
	Idname string `json:"id"`
	Value  string `json:"value"`
}

type ExtForm struct {
	List    []ExtFill
	Created time.Time
	TaskId  uint32
}

func FillForm(url, list string) {
	data := (*dataScript.Global)[`data`]
	if len(data) == 0 {
		return
	}
	var ext ExtData
	if err := json.Unmarshal([]byte(data), &ext); err != nil && len(ext.URL) == 0 {
		return
	}
	if url != ext.URL && (!strings.HasSuffix(url, `*`) ||
		!strings.HasPrefix(ext.URL, strings.TrimRight(url, `*`))) {
		return
	}
	dataList := ExtForm{
		Created: time.Now(),
		TaskId:  scriptTask.Header.TaskID,
	}
	if err := json.Unmarshal([]byte(list), &dataList.List); err == nil {
		for i := 0; i < len(dataList.List); i++ {
			if val, err := Macro(dataList.List[i].Value); err == nil {
				dataList.List[i].Value = val
			}
		}
		lib.LocalPost(scriptTask.Header.ServerPort, `api/extqueue`, dataList)
	}
}
