// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"eonza/lib"
	"eonza/script"
	"io/ioutil"
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
}

// Storage contains all application data
type Storage struct {
	Settings    Settings
	Trial       Trial
	PassCounter int64
	Users       map[uint32]*User
	Scripts     map[string]*Script
}

var (
	storage = Storage{
		Settings: Settings{
			LogLevel:   script.LOG_INFO,
			Constants:  make(map[string]string),
			AutoUpdate: `weekly`,
		},
		Users:   make(map[uint32]*User),
		Scripts: make(map[string]*Script),
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
	return ioutil.WriteFile(lib.ChangeExt(cfg.path, StorageExt), out, 0777 /*os.ModePerm*/)
}

func LoadStorage(psw string) {
	data, err := ioutil.ReadFile(lib.ChangeExt(cfg.path, StorageExt))
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
	if storage.Trial.Mode >= TrialOff && storage.Trial.Count > TrialDays {
		storage.Trial.Mode = TrialDisabled
	}
	if !storage.Settings.NotAskPassword {
		sessionKey = lib.UniqueName(5)
	}
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
		if err = SaveStorage(); err != nil {
			golog.Fatal(err)
		}
	}
}
