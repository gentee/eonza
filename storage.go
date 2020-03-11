// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"eonza/lib"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/kataras/golog"
)

const (
	StorageExt = `eox`
)

// Setting contains settings of the application
/*type Settings struct {
	Lang string // the language of the interface
}*/

// Storage contains all application data
type Storage struct {
	//	Settings   Settings
	Users   []User
	Scripts []Script
	//	History    [HistoryLimit]string
	//	HistoryOff int
}

var (
	storage = Storage{
		/*		Settings: Settings{
				Lang: appInfo.Lang,
			},*/
		Users:   []User{},
		Scripts: []Script{},
	}
	mutex = &sync.RWMutex{}
)

// SaveStorage saves application data
func SaveStorage() error {
	var (
		data bytes.Buffer
		buf  bytes.Buffer
		err  error
	)
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(storage); err != nil {
		return err
	}
	zw := gzip.NewWriter(&buf)
	zw.Name = "data"
	zw.Comment = ""
	zw.ModTime = time.Now()
	_, err = zw.Write(data.Bytes())
	if err != nil {
		return err
	}
	if err = zw.Close(); err != nil {
		return err
	}

	return ioutil.WriteFile(lib.ChangeExt(cfg.path, StorageExt), buf.Bytes(), 0777 /*os.ModePerm*/)
}

func LoadStorage() {
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
	fmt.Println(`STORAGE`, storage)
}
