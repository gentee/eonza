// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"encoding/json"
)

const (
	RF_TEXT     = iota // txt
	RF_MARKDOWN        // md
	RF_HTML            // html
)

type Report struct {
	Title  string `json:"title"`
	Format int64  `json:"format"`
	Body   string `json:"body"`
}

var (
	ReportExt = []string{`txt`, `md`, `html`}

//	reportMutex = sync.Mutex{}
)

func CreateReport(title, body string, format int64) {
	//	reportMutex.Lock()
	//	defer reportMutex.Unlock()
	dataScript.chReport <- Report{
		Title:  title,
		Body:   body,
		Format: format,
	}
}

func ReportToHtml(report Report) (string, error) {
	switch report.Format {
	case RF_MARKDOWN:
	case RF_HTML:
	default:
		report.Body = `<pre>` + report.Body + `</pre>`
	}
	ret, err := json.Marshal(report)
	if err != nil {
		return ``, err
	}
	return string(ret), err
}
