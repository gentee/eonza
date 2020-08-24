// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"eonza/lib"
	"fmt"
	"net/http"
	"strings"

	es "eonza/script"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v2"
)

type ScriptItem struct {
	Name     string           `json:"name"`
	Title    string           `json:"title"`
	Desc     string           `json:"desc,omitempty"`
	Help     string           `json:"help,omitempty"`
	HelpLang string           `json:"helplang,omitempty"`
	Unrun    bool             `json:"unrun,omitempty"`
	Embedded bool             `json:"embedded,omitempty"`
	Folder   bool             `json:"folder,omitempty"`
	Params   []es.ScriptParam `json:"params,omitempty"`
	Initial  string           `json:"initial,omitempty"`
}

type ScriptResponse struct {
	Script
	LangTitle string       `json:"langtitle"`
	Original  string       `json:"original"`
	History   []ScriptItem `json:"history,omitempty"`
	Error     string       `json:"error,omitempty"`
}

type ListResponse struct {
	Map   map[string]ScriptItem `json:"map,omitempty"`
	List  []ScriptItem          `json:"list,omitempty"`
	Cache int32                 `json:"cache"`
	Error string                `json:"error,omitempty"`
}

var (
	hotVersion int32 = 1
)

func deleteScriptHandle(c echo.Context) error {
	var response Response

	if err := DeleteScript(c, c.QueryParam(`name`)); err != nil {
		response.Error = fmt.Sprint(err)
	} else {
		response.Success = true
		hotVersion++
	}
	return c.JSON(http.StatusOK, response)
}

func getScriptHandle(c echo.Context) error {
	var response ScriptResponse

	name := c.QueryParam(`name`)
	if len(name) == 0 {
		name = LatestHistoryEditor(c)
		if len(name) == 0 {
			name = `new`
		}
	}
	idLang := GetLangId(c.(*Auth).User)
	script := getScript(name)
	if script == nil {
		response.Error = Lang(idLang, `erropen`, name)
	} else {
		response.Script = *script
		if response.Script.Settings.Name == `new` {
			response.Script.Settings.Name = lib.UniqueName(7)
			response.Script.Settings.Title = Lang(idLang, `newscript`)
		} else {
			AddHistoryEditor(c.(*Auth).User.ID, script.Settings.Name)
			response.Original = name
			response.LangTitle = es.ReplaceVars(response.Script.Settings.Title,
				script.Langs[c.(*Auth).Lang], &langRes[idLang])
		}
		response.History = GetHistoryEditor(c)
	}
	return c.JSON(http.StatusOK, &response)
}

func saveScriptHandle(c echo.Context) error {
	var (
		script ScriptResponse
		err    error
	)
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = c.Bind(&script); err != nil {
		return errResult()
	}
	if err = script.Validate(); err != nil {
		return errResult()
	}
	if len(script.Original) == 0 {
		if err = AddHistoryEditor(c.(*Auth).User.ID, script.Settings.Name); err != nil {
			return errResult()
		}
	}
	if err = (&script.Script).SaveScript(c, script.Original); err != nil {
		return errResult()
	}
	hotVersion++
	return c.JSON(http.StatusOK, Response{Success: true})
}

func copyParams(src []es.ScriptParam, values map[string]string,
	glob *map[string]string) []es.ScriptParam {
	params := make([]es.ScriptParam, len(src))
	for i, item := range src {
		tmp := item
		tmp.Options.Items = make([]es.ScriptItem, len(tmp.Options.Items))
		for i, val := range item.Options.Items {
			tmp.Options.Items[i] = val
			tmp.Options.Items[i].Title = es.ReplaceVars(val.Title, values, glob)
		}
		tmp.Options.List = copyParams(item.Options.List, values, glob)
		for i, out := range tmp.Options.Output {
			tmp.Options.Output[i] = es.ReplaceVars(out, values, glob)
		}
		params[i] = tmp
		params[i].Title = es.ReplaceVars(params[i].Title, values, glob)
	}
	return params
}

func ScriptToItem(c echo.Context, script *Script) ScriptItem {
	lang := c.(*Auth).Lang
	glob := &langRes[GetLangId(c.(*Auth).User)]
	return ScriptItem{
		Name:     script.Settings.Name,
		Title:    es.ReplaceVars(script.Settings.Title, script.Langs[lang], glob),
		Desc:     es.ReplaceVars(script.Settings.Desc, script.Langs[lang], glob),
		Unrun:    script.Settings.Unrun,
		Help:     script.Settings.Help,
		HelpLang: script.Settings.HelpLang,
		Embedded: script.embedded,
		Folder:   script.folder,
		Params:   copyParams(script.Params, script.Langs[lang], glob),
		Initial:  script.initial,
	}

}

func listScriptHandle(c echo.Context) error {
	resp := &ListResponse{
		Cache: hotVersion,
	}

	if c.QueryParam(`cache`) != fmt.Sprint(hotVersion) {
		list := make(map[string]ScriptItem)

		for _, item := range scripts {
			list[item.Settings.Name] = ScriptToItem(c, item)
		}
		resp.Map = list
	}
	return c.JSON(http.StatusOK, resp)
}

func listRunHandle(c echo.Context) error {
	list := make([]ScriptItem, 0)
	userId := c.(*Auth).User.ID
	if _, ok := userSettings[userId]; !ok {
		return jsonError(c, Lang(DefLang, `unknownuser`, userId))
	}
	for _, name := range userSettings[userId].History.Run {
		if item := getScript(name); item != nil {
			list = append(list, ScriptToItem(c, item))
		}
	}
	return c.JSON(http.StatusOK, &ListResponse{
		List: list,
	})
}

func exportHandle(c echo.Context) error {
	var response Response

	name := c.QueryParam(`name`)
	script := getScript(name)
	if script == nil {
		response.Error = Lang(DefLang, `erropen`, name)
	} else {
		data, err := yaml.Marshal(script)
		if err != nil {
			response.Error = err.Error()
		} else {
			c.Response().Header().Set(echo.HeaderContentDisposition,
				fmt.Sprintf("attachment; filename=%s.yaml", script.Settings.Name))
			//http.ServeContent(c.Response(), c.Request(), "ok.yaml", time.Now(), bytes.NewReader(data))
			return c.Blob(http.StatusOK, "text/yaml", data)
		}
	}
	return c.JSON(http.StatusOK, response)
}

func importHandle(c echo.Context) error {
	var (
		count                          int
		pscript                        *Script
		response                       ScriptResponse
		errFormat, errExists, errEmbed []string
	)
	overwrite := c.FormValue("overwrite") == `true`

	form, err := c.MultipartForm()
	if err != nil {
		return jsonError(c, err)
	}
	files := form.File["files"]

	for _, file := range files {
		src, err := file.Open()
		if err == nil {
			buf := bytes.NewBuffer([]byte{})
			_, err = buf.ReadFrom(src)
			src.Close()
			if err == nil {
				var script Script
				if err = yaml.Unmarshal(buf.Bytes(), &script); err == nil {
					cur := getScript(script.Settings.Name)
					if cur != nil {
						if cur.embedded {
							errEmbed = append(errEmbed, file.Filename)
							continue
						}
						if !overwrite {
							errExists = append(errExists, file.Filename)
							continue
						}
					}
					if err = setScript(&script); err == nil {
						storage.Scripts[lib.IdName(script.Settings.Name)] = &script
						pscript = &script
						count++
					}
				}
			}
		}
		if err != nil {
			errFormat = append(errFormat, file.Filename)
		}
	}
	if count > 0 {
		SaveStorage()
		hotVersion++
		response.Script = *pscript
		response.Original = pscript.Settings.Name
		AddHistoryEditor(c.(*Auth).User.ID, pscript.Settings.Name)
		response.History = GetHistoryEditor(c)
	}
	if len(errFormat) > 0 {
		response.Error = fmt.Sprintf(`Invalid format: %s `, strings.Join(errFormat, `, `))
	}
	if len(errEmbed) > 0 {
		response.Error += fmt.Sprintf(`Can't update embedded scripts: %s `,
			strings.Join(errEmbed, `, `))
	}
	if len(errExists) > 0 {
		response.Error = fmt.Sprintf(`Can't update the existing scripts: %s`,
			strings.Join(errExists, `, `))
	}
	return c.JSON(http.StatusOK, response)
}
