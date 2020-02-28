// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

// LogConfig stores config  settings
type LogConfig struct {
	Dir   string `yaml:"dir"`   // Directory for log files. If it is empty - dir of cfg file
	Mode  string `yaml:"mode"`  // Log mode. It can be stdout, file, stdout file.
	Level string `yaml:"level"` // Log level. It can be disable, error, warn, info.
}

// Config stores application's settings
type Config struct {
	Version   string         `yaml:"version"`   // Version of the application
	Develop   bool           `yaml:"develop"`   // Developer's mode
	AssetsDir string         `yaml:"assetsdir"` // Directory for assets file. empty - dir of cfg file
	Log       LogConfig      `yaml:"log"`       // Log settings
	HTTP      lib.HTTPConfig `yaml:"http"`      // Web-server settings

	path string // path to cfg file
}

var (
	cfg = Config{
		Version: Version,
		Log: LogConfig{
			Mode:  logModeFile,
			Level: logLevelInfo,
		},
		HTTP: lib.HTTPConfig{
			Port:  DefPort,
			Open:  true,
			Theme: `default`,
		},
	}
)

func defDir(dir, def string) string {
	if len(dir) == 0 {
		return filepath.Join(filepath.Dir(cfg.path), def)
	}
	return lib.AppPath(dir)
}

// LoadConfig loads application's settings
func LoadConfig() {
	var (
		err     error
		cfgData []byte
	)

	app := lib.AppPath()
	basename := filepath.Base(app)
	dir := filepath.Dir(app)
	if ext := filepath.Ext(app); len(ext) > 0 {
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
	if len(cfg.AssetsDir) != 0 {
		if _, err := os.Stat(cfg.AssetsDir); err != nil {
			golog.Fatal(err)
		}
	}
	cfg.AssetsDir = defDir(cfg.AssetsDir, DefAssets)
	cfg.Log.Dir = defDir(cfg.Log.Dir, DefLog)
	//	dataFile := defDir(cfg.DataDir)

	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = DefPort
	}
	if len(cfg.HTTP.Theme) == 0 {
		cfg.HTTP.Theme = DefTheme
	}
	SetLogging(basename)
}

// Install creates config and data file on the first execution
func Install() {
	if err := SaveConfig(); err != nil {
		golog.Fatal(err)
	}
	if err := SaveStorage(); err != nil {
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
