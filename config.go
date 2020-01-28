// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

// Config stores application's settings
type Config struct {
	DataDir string `yaml:"datadir"` // Directory for data file. If it is empty - dir of cfg file
	LogDir  string `yaml:"logdir"`  // Directory for log files. If it is empty - dir of cfg file

	path string // Directory of cfg file
}

var (
	cfg = Config{}
)

func defDir(dir string) string {
	var err error
	if len(dir) == 0 {
		dir = filepath.Dir(cfg.path)
	} else if !filepath.IsAbs(dir) {
		if dir, err = filepath.Abs(dir); err != nil {
			golog.Fatal(err)
		}
	}
	return dir
}

// LoadConfig loads application's settings
func LoadConfig() {
	var (
		err     error
		cfgData []byte
	)

	appname := os.Args[0]
	if !filepath.IsAbs(appname) {
		if appname, err = filepath.Abs(appname); err != nil {
			golog.Fatal(err)
		}
	}
	basename := filepath.Base(appname)
	dir := filepath.Dir(appname)
	if ext := filepath.Ext(appname); len(ext) > 0 {
		basename = basename[:len(basename)-len(ext)]
	}
	if len(cfg.path) == 0 {
		cfg.path = filepath.Join(dir, basename+`.yaml`)
		if _, err = os.Stat(cfg.path); os.IsNotExist(err) {
			Install()
		}
	} else if !filepath.IsAbs(cfg.path) {
		if cfg.path, err = filepath.Abs(cfg.path); err != nil {
			golog.Fatal(err)
		}
	}
	if cfgData, err = ioutil.ReadFile(cfg.path); err != nil {
		golog.Fatal(err)
	}
	if err = yaml.Unmarshal(cfgData, &cfg); err != nil {
		golog.Fatal(err)
	}
	dataFile := defDir(cfg.DataDir)
	logFile := defDir(cfg.LogDir)

	fmt.Println(`DIR`, dir, basename, cfg, dataFile, logFile)
}

// Install creates config and data file on the first execution
func Install() {
	if err := SaveConfig(); err != nil {
		golog.Fatal(err)
	}
}

// SaveConfig saves application's settings
func SaveConfig() error {

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cfg.path, data, os.ModePerm)
}
