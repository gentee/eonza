// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"html/template"

	"github.com/kataras/golog"
)

type Render struct {
	App     AppInfo
	Version string
	Develop bool
	//	Port    int
	/*	Params   map[string]string
		Url      string
		Index    bool
	*/
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
		"lang": Lang,
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

func RenderPage(url string) (string, error) {
	var (
		err    error
		render Render
	)

	/*	file := filepath.Join(cfg.WebDir, filepath.FromSlash(page.url))
		var exist bool
		if cfg.mode != ModeDynamic {
			if _, err := os.Stat(file); err == nil {
				exist = true
			}
		}
		switch cfg.mode {
		case ModeLive:
		case ModeCache:
		case ModeStatic:
			if !exist {
				return ``, ErrNotFound
			}
		}
		if exist {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				return ``, err
			}
			return string(data), nil
		}
		if len(page.Template) == 0 {
			page.Template = page.parent.Template
		}
		tpl := page.Template
		if len(tpl) == 0 {
			return page.body, err
		}
		render.Content = template.HTML(``)*/
	render.App = appInfo
	render.Version = Version
	render.Develop = cfg.Develop

	buf := bytes.NewBuffer([]byte{})
	if err = tmpl.ExecuteTemplate(buf, url, render); err != nil {
		return ``, err
	}
	return buf.String(), err
}
