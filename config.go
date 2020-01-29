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

// HTTPConfig stores web-server settings
type HTTPConfig struct {
	Port int `yaml:"port"` // if empty, then 3234
}

// LogConfig stores config  settings
type LogConfig struct {
	Dir   string `yaml:"dir"`   // Directory for log files. If it is empty - dir of cfg file
	Mode  string `yaml:"mode"`  // Log mode. It can be stdout, file, stdout file.
	Level string `yaml:"level"` // Log level. It can be disable, error, warn, info.
}

// Config stores application's settings
type Config struct {
	Version string     `yaml:"version"` // Version of the application
	DataDir string     `yaml:"datadir"` // Directory for data file. If it is empty - dir of cfg file
	Log     LogConfig  `yaml:"log"`     // Log settings
	HTTP    HTTPConfig `yaml:"http"`    // Web-server settings

	path string // Directory of cfg file
}

var (
	cfg = Config{
		Version: Version,
		Log: LogConfig{
			Mode:  logModeFile,
			Level: logLevelInfo,
		},
		HTTP: HTTPConfig{
			Port: DefPort,
		},
	}
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

	SetLogging(basename)
	fmt.Println(`DIR`, dir, basename, cfg, dataFile)
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
	return ioutil.WriteFile(cfg.path, data, 0777 /*os.ModePerm*/)
}
