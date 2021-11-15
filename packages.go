// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"eonza/lib"
	es "eonza/script"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v2"
)

type PackageInfo struct {
	Title    string `json:"title" yaml:"title"`
	Desc     string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Help     string `json:"help,omitempty" yaml:"help,omitempty"`
	HelpLang string `json:"helplang,omitempty" yaml:"helplang,omitempty"`
	// Calculated fields
	Installed bool `json:"installed"`
}

type PackageReview struct {
	PackageInfo
	Name      string `json:"name"`
	HasParams bool   `json:"hasparams"`
}

type PackagesResponse struct {
	List  []PackageReview `json:"list,omitempty"`
	Error string          `json:"error,omitempty"`
}

type PackageResponse struct {
	Params []es.ScriptParam       `json:"params,omitempty"`
	Values map[string]interface{} `json:"values,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

type Package struct {
	PackageInfo `yaml:"info"`
	Version     string                       `yaml:"version"`
	Langs       map[string]map[string]string `json:"langs,omitempty" yaml:"langs,omitempty"`
	Params      []es.ScriptParam             `json:"params,omitempty" yaml:"params,omitempty"`

	json string // json of package values
}

func GetPackageJSON(name string) string {
	pkg := Assets.Packages[name]
	if pkg == nil || !pkg.Installed || len(pkg.Params) == 0 {
		return ``
	}
	if len(pkg.json) == 0 {
		if out, err := json.Marshal(storage.PkgValues[name]); err == nil {
			pkg.json = string(out)
		}
	}
	return pkg.json
}

func LoadPackageScripts(name string) {
	isfolder := func(script *Script) bool {
		return script.Settings.Name == SourceCode ||
			strings.Contains(script.Code, `%body%`)
	}
	for _, f := range PackagesFS.List {
		if strings.HasPrefix(f.Name, name) && filepath.Ext(f.Name) == `.yaml` &&
			filepath.Base(f.Name) != `package.yaml` {
			var script Script
			if err := yaml.Unmarshal(f.Data, &script); err != nil {
				golog.Error(err)
			}
			script.embedded = true
			script.folder = isfolder(&script)
			script.pkg = name
			if err := setScript(&script); err != nil {
				golog.Error(err)
			}
		}
	}
	hotVersion++
}

func UnloadPackageScripts(c echo.Context, name string) error {
	todel := make(map[string]bool)

	for _, f := range PackagesFS.List {
		if strings.HasPrefix(f.Name, name) && filepath.Ext(f.Name) == `.yaml` &&
			filepath.Base(f.Name) != `package.yaml` {
			sname := filepath.Base(f.Name)
			todel[sname[:len(sname)-5]] = true
		}
	}

	for key := range todel {
		if script := getScript(key); script != nil {
			if err := checkDep(c, key, script.Settings.Title); err != nil {
				return err
			}
		}
	}
	for key := range todel {
		delScript(key)
	}
	hotVersion++
	return nil
}

func LoadPackages() {
	for name, ext := range Assets.Packages {
		if _, err := os.Stat(filepath.Join(cfg.PackagesDir, name)); err == nil {
			ext.Installed = true
			LoadPackageScripts(name)
			Assets.Packages[name] = ext
		}
	}
}

func PackagesList(c echo.Context) *PackagesResponse {
	lang := c.(*Auth).Lang
	glob := &langRes[GetLangId(c.(*Auth).User)]
	ret := make([]PackageReview, 0)
	for name, pkg := range Assets.Packages {
		pkg.Title = es.ReplaceVars(pkg.Title, pkg.Langs[lang], glob)
		pkg.Desc = es.ReplaceVars(pkg.Desc, pkg.Langs[lang], glob)
		ret = append(ret, PackageReview{
			PackageInfo: pkg.PackageInfo,
			Name:        name,
			HasParams:   len(pkg.Params) > 0,
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Installed == ret[j].Installed {
			return ret[i].Title < ret[j].Title
		}
		return ret[i].Installed
	})
	return &PackagesResponse{
		List: ret,
	}
}

func packagesHandle(c echo.Context) error {
	if err := CheckAdmin(c); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, PackagesList(c))
}

func findPackage(c echo.Context, name string) (*Package, error) {
	if err := CheckAdmin(c); err != nil {
		return nil, err
	}
	v, ok := Assets.Packages[name]
	if !ok {
		return nil, fmt.Errorf(`Cannot find %s package`, name)
	}
	return v, nil
}

func packageHandle(c echo.Context) error {
	var (
		err error
		pkg *Package
	)
	name := c.Param("name")
	if pkg, err = findPackage(c, name); err != nil {
		return jsonError(c, err)
	}
	ret := make([]es.ScriptParam, len(pkg.Params))
	copy(ret, pkg.Params)
	lang := c.(*Auth).Lang
	glob := &langRes[GetLangId(c.(*Auth).User)]
	for i, par := range ret {
		ret[i].Title = es.ReplaceVars(par.Title, pkg.Langs[lang], glob)
	}
	return c.JSON(http.StatusOK, &PackageResponse{
		Params: ret,
		Values: storage.PkgValues[name],
	})
}

func savePackageHandle(c echo.Context) error {
	var (
		err error
		pkg *Package
	)

	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = CheckAdmin(c); err != nil {
		return errResult()
	}
	name := c.Param("name")
	if pkg, err = findPackage(c, name); err != nil {
		return jsonError(c, err)
	}
	values := make(map[string]interface{})
	if err = c.Bind(&values); err != nil {
		return errResult()
	}
	delete(values, `name`)
	storage.PkgValues[name] = values
	pkg.json = ""
	if err = SaveStorage(); err != nil {
		return errResult()
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}

func packageInstallHandle(c echo.Context) error {
	var (
		err error
		pkg *Package
	)
	if cfg.playground {
		return jsonError(c, fmt.Errorf(`Access denied`))
	}
	name := c.Param("name")
	if pkg, err = findPackage(c, name); err != nil {
		return jsonError(c, err)
	}
	if name == `tests` {
		storage.Events[`test`] = &Event{
			ID:        lib.RndNum(),
			Name:      `test`,
			Script:    `data-print`,
			Token:     `TEST_TOKEN`,
			Whitelist: `::1/128, 127.0.0.0/31`,
			Active:    true,
		}
	}
	vals := make(map[string]interface{})
	cur := storage.PkgValues[name]
	for _, par := range pkg.Params {
		vals[par.Name] = par.Options.Initial
		if cur != nil {
			if v, ok := cur[par.Name]; ok {
				vals[par.Name] = v
			}
		}
	}
	if len(vals) > 0 {
		storage.PkgValues[name] = vals
		if err = SaveStorage(); err != nil {
			return jsonError(c, err)
		}
	}
	path := filepath.Join(cfg.PackagesDir, name)
	if err := os.MkdirAll(path, 0777); err != nil {
		return jsonError(c, err)
	}
	for _, f := range PackagesFS.List {
		if strings.HasPrefix(f.Name, name+`/files`) {
			fullName := filepath.Join(cfg.PackagesDir, f.Name)
			if f.Dir {
				if err := os.MkdirAll(fullName, 0777); err != nil {
					os.RemoveAll(path)
					return jsonError(c, err)
				}
			} else if err = os.WriteFile(fullName, f.Data, 0666); err != nil {
				os.RemoveAll(path)
				return jsonError(c, err)
			}
		}
	}
	LoadPackageScripts(name)
	pkg.Installed = true
	return c.JSON(http.StatusOK, PackagesList(c))
}

func packageUninstallHandle(c echo.Context) error {
	var (
		err error
		pkg *Package
	)
	if cfg.playground {
		return jsonError(c, fmt.Errorf(`Access denied`))
	}
	name := c.Param("name")
	if pkg, err = findPackage(c, name); err != nil {
		return jsonError(c, err)
	}
	if err = UnloadPackageScripts(c, name); err != nil {
		return jsonError(c, err)
	}
	path := filepath.Join(cfg.PackagesDir, name)
	if err := os.RemoveAll(path); err != nil {
		return jsonError(c, err)
	}
	pkg.Installed = false
	if name == `tests` {
		delete(storage.Events, `test`)
		SaveStorage()
	}
	return c.JSON(http.StatusOK, PackagesList(c))
}

func PkgFile(assetname, filename string) error {
	data, err := os.ReadFile(filepath.Join(scriptTask.Header.PkgPath, assetname))
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0666)
}
