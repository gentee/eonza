// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"eonza/lib"
	"eonza/script"
	"eonza/users"
	"fmt"
	"html"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
)

type Render struct {
	App         AppInfo
	AppPath     string
	Version     string
	CompileDate string
	Title       string
	Develop     bool
	Playground  bool
	Tray        bool
	Langs       map[string]string
	LangRes     map[string]map[string]string
	Lang        string
	Login       bool
	Localhost   bool
	Favs        []Fav
	Nfy         *NfyResponse
	//	Update      VerUpdate
	Pro       bool
	ProActive bool
	DefLists  []DefList
	User      *users.User
	Twofa     bool
	AskMaster bool
	//	ProSettings ProSettings
	//	Port    int
	/*	Params   map[string]string
		Url      string
		Index    bool
	*/
}

type RenderScript struct {
	Task
	Title      string
	IsScript   bool
	IsAutoFill bool
	URLPort    int
	Start      string
	Finish     string
	CDN        string
	Nickname   string
	Role       string
	Source     template.HTML
	Stdout     template.HTML
	Logout     template.HTML
	Reports    []script.Report
	FormAlign  uint32
}

var (
	tmpl *template.Template
)

func Html(par string) template.HTML {
	return template.HTML(par)
}

func Time2Str(t time.Time) string {
	if t.Year() < 1900 {
		return ``
	}
	return t.Format(TimeFormat)
}

func InitTemplates() {
	var err error
	tmpl = template.New(`assets`).Delims(`[[`, `]]`).Funcs(template.FuncMap{
		"html":     Html,
		"time2str": Time2Str,
	})
	for _, fname := range Assets.Templates {
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
		var out string
		if len(strings.TrimSpace(input)) != 0 {
			out = strings.ReplaceAll(html.EscapeString(input), "\n", `<br>`)
			if isLog {
				for key, item := range map[string]string{`INFO`: `egreen`, `FORM`: `eblue`,
					`WARN`: `eyellow`, `ERROR`: `ered`} {
					out = strings.ReplaceAll(out,
						"["+key+"]", fmt.Sprintf(`<span class="%s">[%s]</span>`,
							item, key))
				}
			}
		}
		return template.HTML(out)
	}
	if url == `script` {
		if IsScript {
			renderScript.Task = task
			renderScript.Title = scriptTask.Header.Title
			renderScript.CDN = scriptTask.Header.CDN
			renderScript.Nickname = scriptTask.Header.User.Nickname
			renderScript.Role = scriptTask.Header.Role.Name
			renderScript.URLPort = scriptTask.Header.URLPort
			renderScript.IsAutoFill = scriptTask.Header.IsAutoFill
			renderScript.FormAlign = scriptTask.Header.FormAlign
		} else {
			renderScript.URLPort = cfg.HTTP.Port
			renderScript.Task = *c.Get(`Task`).(*Task)
			renderScript.Title = c.Get(`Title`).(string)
			files, replist := GetTaskFiles(renderScript.Task.ID, true)
			renderScript.Stdout = out2html(files[TExtOut], false)
			renderScript.Logout = out2html(files[TExtLog], true)
			renderScript.Reports = replist
			renderScript.Task.SourceCode = files[TExtSrc]
			renderScript.Nickname, renderScript.Role = GetUserRole(renderScript.Task.UserID, renderScript.Task.RoleID)
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
		user := c.(*Auth).User
		render.App = appInfo
		render.AppPath = strings.Join(os.Args, ` `)
		render.Version = GetVersion()
		render.CompileDate = CompileDate
		render.Title = storage.Settings.Title
		render.Develop = cfg.develop
		render.Playground = cfg.playground
		render.Tray = isTray
		render.Langs = make(map[string]string)
		if c.Request().URL.Path == `install` {
			render.LangRes = make(map[string]map[string]string)
			render.Lang = LangDefCode
			for _, item := range strings.Split(c.Request().Header.Get(`Accept-Language`), `,`) {
				if len(item) >= 2 {
					if _, ok := langsId[item[:2]]; ok {
						render.Lang = item[:2]
						break
					}
				}
			}
			for i, lang := range langs {
				render.LangRes[lang] = make(map[string]string)
				for _, key := range []string{`continue`, `sellang`} {
					v := langRes[i][key]
					if len(v) == 0 {
						v = langRes[0][key]
					}
					render.LangRes[lang][key] = v
				}
			}
		} else {
			render.Lang = GetLangCode(user)
		}
		for i, lang := range langs {
			render.Langs[lang] = Lang(i, `native`)
		}

		render.Login = len(storage.Settings.PasswordHash) > 0
		render.Localhost = cfg.HTTP.Host == Localhost
		render.Favs = FilterFavs(userSettings[user.ID].Favs)
		render.Nfy = NfyList(false, user.ID, user.RoleID)
		//		render.Update = nfyData.Update
		//		render.Update.Notify = GetNewVersion(GetLangCode(c.(*Auth).User))
		render.Pro = Pro && !cfg.playground
		render.ProActive = IsProActive()
		render.DefLists = defaultList
		render.User = c.(*Auth).User
		render.Twofa = IsTwofa()
		render.AskMaster = IsProActive() && !IsDecrypted()
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
