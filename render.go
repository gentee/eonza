// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"html/template"
)

type Render struct {
	Content template.HTML
	/*	Title    string
		Params   map[string]string
		Url      string
		Index    bool
	*/
}

func RenderPage(url string) (string, error) {
	var (
		err error
		//		ok     bool
		//		render Render
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
	/*	render.Title = page.Title
		render.Params = page.parent.Params
		render.Langs = LangList(page)
		render.Index = path.Base(page.url) == `index.html`
		render.Url = page.url
		render.Domain = cfg.Domain*/
	//	render.Original = path.Join(path.Dir(page.url), path.Base(page.file))
	/*	if err = templates.templates.ExecuteTemplate(buf, tpl+`.html`, render); err != nil {
		return ``, err
	}*/
	//buf := bytes.NewBuffer([]byte{})
	return `Hello` /*buf.String()*/, err
}
