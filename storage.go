// Copyright 2020-21 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"eonza/lib"
	"eonza/script"
	"os"
	"sync"
	"time"

	"github.com/kataras/golog"
	"golang.org/x/crypto/bcrypt"
)

const (
	StorageExt = `eox`
	TrialDays  = 30

	TrialDisabled = -1
	TrialOff      = 0
	TrialOn       = 1

	DefMaxTasks    = 100
	DefRemoveAfter = 14
)

type Trial struct {
	Mode  int       `json:"mode"`
	Count int       `json:"count"`
	Last  time.Time `json:"last"`
}

// Setting contains settings of the application
type Settings struct {
	LogLevel       int               `json:"loglevel"`
	IncludeSrc     bool              `json:"includesrc"`
	Constants      map[string]string `json:"constants"`
	PasswordHash   []byte            `json:"passwordhash"`
	NotAskPassword bool              `json:"notaskpassword"`
	Title          string            `json:"title"`
	HideTray       bool              `json:"hidetray"`
	AutoUpdate     string            `json:"autoupdate"`
	RemoveAfter    int               `json:"removeafter"`
	MaxTasks       int               `json:"maxtasks"`
	HideDupTasks   bool              `json:"hideduptasks"`
}

// Storage contains all application data
type Storage struct {
	Version     string
	Settings    Settings
	Trial       Trial
	PassCounter int64
	Users       map[uint32]*User // Deprecated
	Scripts     map[string]*Script
	Timers      map[uint32]*Timer
	Events      map[string]*Event
	Browsers    []*Browser
}

var (
	storage = Storage{
		Version: GetVersion(),
		Settings: Settings{
			LogLevel:    script.LOG_INFO,
			Constants:   make(map[string]string),
			AutoUpdate:  `weekly`,
			MaxTasks:    DefMaxTasks,
			RemoveAfter: DefRemoveAfter,
		},
		Users:    make(map[uint32]*User),
		Scripts:  make(map[string]*Script),
		Timers:   make(map[uint32]*Timer),
		Browsers: make([]*Browser, 0),
		Events:   map[string]*Event{
			/*			`test`: {
						ID:        lib.RndNum(),
						Name:      `test`,
						Script:    `data-print`,
						Token:     `TEST_TOKEN`,
						Whitelist: `::1/128, 127.0.0.0/31`,
						Active:    true,
					},*/
		},
	}
	mutex = &sync.Mutex{}
)

// SaveStorage saves application data
func SaveStorage() error {
	var (
		data bytes.Buffer
		out  []byte
		err  error
	)
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(storage); err != nil {
		return err
	}
	if out, err = lib.GzipCompress(data.Bytes()); err != nil {
		return err
	}
	return os.WriteFile(lib.ChangeExt(cfg.path, StorageExt), out, 0777 /*os.ModePerm*/)
}

func LoadStorage(psw string) {
	data, err := os.ReadFile(lib.ChangeExt(cfg.path, StorageExt))
	if err != nil {
		golog.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		golog.Fatal(err)
	}
	//	if _, err := io.Copy(&buf, zr); err != nil {
	dec := gob.NewDecoder(zr)
	if err = dec.Decode(&storage); err != nil {
		golog.Fatal(err)
	}
	if err := zr.Close(); err != nil {
		golog.Fatal(err)
	}
	if storage.Trial.Mode != TrialDisabled && storage.Trial.Count > TrialDays {
		storage.Trial.Mode = TrialDisabled
	}
	if cfg.playground {
		storage.Trial.Mode = TrialDisabled
	}
	if !storage.Settings.NotAskPassword {
		sessionKey = lib.UniqueName(5)
	}
	var save bool
	if len(psw) > 0 {
		var hash []byte
		if psw != `reset` {
			hash, err = bcrypt.GenerateFromPassword([]byte(psw), 11)
			if err != nil {
				golog.Fatal(err)
			}
		}
		storage.Settings.PasswordHash = hash
		storage.PassCounter++
		save = true
		if err = SaveStorage(); err != nil {
			golog.Fatal(err)
		}
	}
	if storage.Version != GetVersion() {
		Migrate()
		save = true
	}
	if save {
		if err = SaveStorage(); err != nil {
			golog.Fatal(err)
		}
	}
}

func StoragePassCounter() error {
	storage.PassCounter++
	return SaveStorage()
}
