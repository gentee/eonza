// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"html/template"
)

type Render struct {
	Title string
	/*	Params   map[string]string
		Url      string
		Index    bool
	*/
}

var (
	tmpls = make(map[string]*template.Template)
)

func RenderTemplate(name string) (*template.Template, error) {
	var (
		ok   bool
		err  error
		tmpl *template.Template
	)

	if tmpl, ok = tmpls[name]; !ok {
		tmpl = template.New(name).Delims(`[[`, `]]`)
		data := TemplateAsset(name)
		if len(data) == 0 {
			return nil, ErrNotFound
		}
		if tmpl, err = tmpl.Parse(string(data)); err != nil {
			return nil, err
		}
		tmpls[name] = tmpl
	}
	return tmpl, nil
}

func RenderPage(url string) (string, error) {
	var (
		err    error
		render Render
		tmpl   *template.Template
	)
	if tmpl, err = RenderTemplate(url); err != nil {
		return ``, err
	}

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
	render.Title = `My Script`
	/*	render.Params = page.parent.Params
		render.Langs = LangList(page)
		render.Index = path.Base(page.url) == `index.html`
		render.Url = page.url
		render.Domain = cfg.Domain*/
	//	render.Original = path.Join(path.Dir(page.url), path.Base(page.file))
	buf := bytes.NewBuffer([]byte{})
	if err = tmpl.Execute(buf, render); err != nil {
		return ``, err
	}
	return buf.String(), err
}
