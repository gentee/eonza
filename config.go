// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"eonza/users"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v2"
)

const (
	ModeDefault    = `default`
	ModeDevelop    = `develop`
	ModePlayground = `playground`
)

// LogConfig stores config  settings
type LogConfig struct {
	Dir   string `yaml:"dir"`   // Directory for log files. If it is empty - dir of cfg file
	Mode  string `yaml:"mode"`  // Log mode. It can be stdout, file, stdout file.
	Level string `yaml:"level"` // Log level. It can be disable, error, warn, info.
}

// UsersConfig stores the config of users
type UsersConfig struct {
	Dir string `yaml:"dir"` // Directory for users files. If it is empty - dir of cfg file
}

// Config stores application's settings
type Config struct {
	Mode string `yaml:"mode"` // Mode: default, develop, playground
	// Empty dir means subfolder in dir of cfg file
	AssetsDir  string               `yaml:"assetsdir"`           // Directory for assets file.
	ExtsDir    string               `yaml:"extsdir"`             // Directory for extensions files.
	Log        LogConfig            `yaml:"log"`                 // Log settings
	Users      UsersConfig          `yaml:"users"`               // Users settings
	HTTP       lib.HTTPConfig       `yaml:"http"`                // Web-server settings
	Playground lib.PlaygroundConfig `yaml:"playground"`          // Playground settings
	Whitelist  []string             `yaml:"whitelist,omitempty"` // Whitelist of IP-addresses

	path       string // path to cfg file
	develop    bool
	playground bool
}

var (
	cfg = Config{
		Mode: ModeDefault,
		Log: LogConfig{
			Mode:  logModeFile,
			Level: logLevelInfo,
		},
		HTTP: lib.HTTPConfig{
			Host:   Localhost,
			Port:   DefPort,
			Open:   true,
			Theme:  `default`,
			JWTKey: lib.UniqueName(12),
			Cert:   ``,
			Priv:   ``,
		},
	}
	firstRun bool
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
	if cfgData, err = os.ReadFile(cfg.path); err != nil {
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
	//	cfg.AssetsDir = defDir(cfg.AssetsDir, DefAssets)
	cfg.ExtsDir = defDir(cfg.ExtsDir, DefExts)
	cfg.Log.Dir = defDir(cfg.Log.Dir, DefLog)
	cfg.Users.Dir = defDir(cfg.Users.Dir, DefUsers)
	//	dataFile := defDir(cfg.DataDir)
	if len(cfg.HTTP.Host) == 0 {
		cfg.HTTP.Host = Localhost
	} else if cfg.HTTP.Host != Localhost {
		cfg.HTTP.Open = false
	}
	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = DefPort
	}
	if cfg.HTTP.LocalPort == 0 {
		if cfg.HTTP.LocalPort, err = getPort(); err != nil {
			golog.Fatal(err)
		}
	}
	if len(cfg.HTTP.Theme) == 0 {
		cfg.HTTP.Theme = DefTheme
	}
	if cfg.HTTP.Host != Localhost {
		if cfg.HTTP.Cert == nil {
			golog.Fatal(`Specify the path to the certificate pem file in config file`)
		}
		if cfg.HTTP.Priv == nil {
			golog.Fatal(`Specify the path to the private key pem file in config file`)
		}
		if cfg.HTTP.Cert, err = os.ReadFile(fmt.Sprint(cfg.HTTP.Cert)); err != nil {
			golog.Fatal(err)
		}
		if cfg.HTTP.Priv, err = os.ReadFile(fmt.Sprint(cfg.HTTP.Priv)); err != nil {
			golog.Fatal(err)
		}
	}
	cfg.develop = cfg.Mode == ModeDevelop
	cfg.playground = cfg.Mode == ModePlayground
	if cfg.playground {
		if cfg.Playground.Tasks == 0 {
			cfg.Playground.Tasks = DefTaskLimit
		}
	}

	SetLogging(basename)
	if err = InitTaskManager(); err != nil {
		golog.Fatal(err)
	}
}

// Install creates config and data file on the first execution
func Install() {
	var err error

	firstRun = true
	scripts = make(map[string]*Script)
	InitAssets()
	for _, f := range InitFS.Files {
		var script Script
		if f.Dir {
			continue
		}
		if err := yaml.Unmarshal(f.Data, &script); err != nil {
			golog.Fatal(err)
		}
		if err := setScript(&script); err != nil {
			golog.Fatal(err)
		}
		storage.Scripts[lib.IdName(script.Settings.Name)] = &script
	}
	if err = SaveConfig(); err != nil {
		golog.Fatal(err)
	}
	cfg.Users.Dir = defDir(cfg.Users.Dir, DefUsers)
	err = os.MkdirAll(cfg.Users.Dir, 0777)
	if err != nil {
		golog.Fatal(err)
	}
	ProInit(nil, 0)
	userSettings[users.XRootID] = UserSettings{
		ID:   users.XRootID,
		Lang: appInfo.Lang,
		Favs: []Fav{
			{Name: `welcome`},
			{Name: `Tools`, IsFolder: true, Children: []Fav{
				{Name: `copy-files`},
				{Name: `create-archive`},
			}},
		},
	}
	if err = SaveUser(users.XRootID); err != nil {
		golog.Fatal(err)
	}
	if err = SaveStorage(); err != nil {
		golog.Fatal(err)
	}
}

// SaveConfig saves application's settings
func SaveConfig() error {

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.path, data, 0777 /*os.ModePerm*/)
}
