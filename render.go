// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"eonza/lib"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

type Render struct {
	App     AppInfo
	Version string
	Develop bool
	Langs   map[string]string
	//	Port    int
	/*	Params   map[string]string
		Url      string
		Index    bool
	*/
}

type RenderScript struct {
	Task
	Title    string
	IsScript bool
	Start    string
	Finish   string
	Source   template.HTML
	Stdout   template.HTML
	Logout   template.HTML
}

var (
	tmpl *template.Template
)

func Html(par string) template.HTML {
	return template.HTML(par)
}

func InitTemplates() {
	var err error
	tmpl = template.New(`assets`).Delims(`[[`, `]]`).Funcs(template.FuncMap{
		"html": Html,
	})
	for _, tpl := range _escDirs["../eonza-assets/themes/default/templates"] {
		fname := tpl.Name()
		fname = fname[:len(fname)-4]
		data := TemplateAsset(fname)
		if len(data) == 0 {
			golog.Fatal(ErrNotFound)
		}
		tmpl = tmpl.New(fname)

		if tmpl, err = tmpl.Parse(string(data)); err != nil {
			golog.Fatal(err)
		}
	}
}

func RenderPage(c echo.Context, url string) (string, error) {
	var (
		err          error
		render       Render
		renderScript RenderScript
		data         interface{}
	)

	out2html := func(input string, isLog bool) template.HTML {
		out := strings.ReplaceAll(input, "\n", `<br>`)
		if isLog {
			for key, item := range map[string]string{`INFO`: `egreen`, `FORM`: `eblue`,
				`WARN`: `eyellow`, `ERROR`: `ered`} {
				out = strings.ReplaceAll(out, "["+key+"]", fmt.Sprintf(`<span class="%s">[%s]</span>`,
					item, key))
			}
		}
		return template.HTML(out)
	}
	if url == `script` {
		if IsScript {
			renderScript.Task = task
			renderScript.Title = scriptTask.Header.Title
		} else {
			renderScript.Task = *c.Get(`Task`).(*Task)
			renderScript.Title = c.Get(`Title`).(string)
			files := GetTaskFiles(renderScript.Task.ID)
			renderScript.Stdout = out2html(files[TExtOut], false)
			renderScript.Logout = out2html(files[TExtLog], true)
			renderScript.Task.SourceCode = files[TExtSrc]
		}
		if len(renderScript.Task.SourceCode) > 0 {
			if out, err := lib.Markdown("```go\r\n" + renderScript.Task.SourceCode +
				"\r\n```"); err == nil {
				renderScript.Source = template.HTML(out)
			}
		}
		renderScript.Start = time.Unix(renderScript.Task.StartTime, 0).Format(TimeFormat)
		if renderScript.FinishTime != 0 && renderScript.Task.Status >= TaskFinished {
			renderScript.Finish = time.Unix(renderScript.Task.FinishTime, 0).Format(TimeFormat)
		}
		renderScript.IsScript = IsScript
		data = renderScript
	} else {
		render.App = appInfo
		render.Version = Version
		render.Develop = cfg.Develop
		render.Langs = make(map[string]string)
		for i, lang := range langs {
			render.Langs[lang] = Lang(i, `native`)
		}
		data = render
	}

	buf := bytes.NewBuffer([]byte{})
	if err = tmpl.ExecuteTemplate(buf, url, data); err != nil {
		return ``, err
	}
	body := buf.String()
	if strings.IndexRune(body, LangChar) != -1 {
		body = RenderLang([]rune(body), GetLangId(c.(*Auth).User))
	}

	return body, err
}
