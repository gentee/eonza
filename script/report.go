// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"eonza/lib"
	"html"
	"os"
	"strings"
)

const (
	RF_TEXT     = iota // txt
	RF_MARKDOWN        // md
	RF_HTML            // html

	ReportExt = `eor`
)

type Report struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

var (
	ReportsExt = []string{`txt`, `md`, `html`}

//	reportMutex = sync.Mutex{}
)

func CreateReport(title, body string) {
	//	reportMutex.Lock()
	//	defer reportMutex.Unlock()
	SetTimeout(500)
	dataScript.chReport <- Report{
		Title: title,
		Body:  strings.TrimSpace(body),
	}
}

func GetReportExt(report Report) string {
	return ReportsExt[ReportType(report)]
}

func ReportType(report Report) int {
	if len(report.Body) == 0 {
		return RF_TEXT
	}
	if report.Body[0] == '<' {
		return RF_HTML
	}
	if strings.HasPrefix(report.Body, `# `) || strings.HasPrefix(report.Body, `## `) ||
		strings.HasPrefix(report.Body, `### `) {
		return RF_MARKDOWN
	}
	return RF_TEXT
}

func ReportToHtml(report Report) string {
	var (
		ret string
		err error
	)
	switch ReportType(report) {
	case RF_MARKDOWN:
		ret, err = lib.Markdown(report.Body)
		if err != nil {
			ret = `<pre>` + err.Error() + `</pre>`
		}
	case RF_HTML:
		ret = report.Body
	default:
		ret = `<pre>` + html.EscapeString(report.Body) + `</pre>`
	}
	return string(ret)
}

func ReportToJSON(report Report) (string, error) {
	report.Body = ReportToHtml(report)
	ret, err := json.Marshal(report)
	if err != nil {
		return ``, err
	}
	return string(ret), err
}

func SaveReport(fname string, list []Report) error {
	var (
		data bytes.Buffer
		err  error
	)
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(list); err != nil {
		return err
	}
	return os.WriteFile(fname, data.Bytes(), 0777 /*os.ModePerm*/)
}
