// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"database/sql"
	"fmt"
	"strings"
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
		return fmt.Errorf(`The '%s' identifier doesn't exist`, name)
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
		return nil, fmt.Errorf(`The '%s' identifier doesn't exist`, name)
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

func SQLGet(varname string) (*sql.DB, error) {
	var (
		db *sql.DB
		ok bool
	)
	val, err := GetHandle(varname)
	if err != nil {
		return nil, err
	}
	if db, ok = val.(*sql.DB); !ok {
		return nil, ErrInvalidPar
	}
	return db, err
}

func SQLClose(varname string) error {
	var (
		db  *sql.DB
		err error
	)
	if db, err = SQLGet(varname); err != nil {
		return ErrInvalidPar
	}
	CloseHandle(varname)
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

func SQLExec(varname string, sqlexec string, pars *core.Array) error {
	var (
		db  *sql.DB
		err error
		buf string
	)
	if db, err = SQLGet(varname); err != nil {
		return ErrInvalidPar
	}
	for _, item := range strings.Split(sqlexec, "\n") {
		tmp := strings.TrimSpace(item)
		if strings.HasPrefix(tmp, `#`) || strings.HasPrefix(tmp, `--`) {
			if len(buf) > 0 {
				if _, err = db.Exec(buf); err != nil {
					return err
				}
				buf = ``
			}
		} else {
			buf += item + "\n"
		}
	}
	if len(buf) > 0 {
		_, err = db.Exec(buf, pars.Data...)
	}
	return err
}
