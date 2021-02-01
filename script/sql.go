// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gentee/gentee/core"
)

var (
	handles     = make(map[string]interface{})
	handleMutex = &sync.Mutex{}
)

func CloseHandle(name string) error {
	handleMutex.Lock()
	defer handleMutex.Unlock()
	if _, ok := handles[name]; !ok {
		return fmt.Errorf(`The '%s' identifier exists`, name)
	}
	delete(handles, name)
	return nil
}

func GetHandle(name string) (interface{}, error) {
	handleMutex.Lock()
	defer handleMutex.Unlock()
	var (
		ret interface{}
		ok  bool
	)
	if ret, ok = handles[name]; !ok {
		return nil, fmt.Errorf(`The '%s' identifier exists`, name)
	}
	return ret, nil
}

func SetHandle(name string, value interface{}) error {
	handleMutex.Lock()
	defer handleMutex.Unlock()

	if _, ok := handles[name]; ok {
		return fmt.Errorf(`The '%s' identifier exists`, name)
	}
	handles[name] = value
	return nil
}

func SQLClose(varname string) error {
	var (
		db *sql.DB
		ok bool
	)
	val, err := GetHandle(varname)
	if err != nil {
		return err
	}
	if db, ok = val.(*sql.DB); !ok {
		return ErrInvalidPar
	}
	return db.Close()
}

func SQLConnection(pars *core.Map, varname string) error {
	var (
		db  *sql.DB
		err error
	)
	spar := pars.Data
	host := spar["host"].(string)
	if len(host) == 0 {
		host = `localhost`
	}
	port := spar["port"].(string)
	if spar["sqlserver"].(string) == `pg` {

	} else {
		if len(port) == 0 {
			port = `3306`
		}
		db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", spar["username"].(string),
			spar["password"].(string), host, port, spar["dbname"].(string)))
		if err != nil {
			return err
		}
		if err = db.Ping(); err != nil {
			return err
		}
	}
	return SetHandle(varname, db)
}
