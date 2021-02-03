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
	_ "github.com/lib/pq"

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
		if len(port) == 0 {
			port = `5432`
		}
		db, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname='%s' sslmode=disable",
			host, port, spar["username"].(string), spar["password"].(string), spar["dbname"].(string)))
	} else {
		if len(port) == 0 {
			port = `3306`
		}
		db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", spar["username"].(string),
			spar["password"].(string), host, port, spar["dbname"].(string)))
	}
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
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

func sqlColumns(rows *sql.Rows) (columns []string, ptrs []interface{}, err error) {
	if columns, err = rows.Columns(); err != nil {
		return
	}
	ptrs = make([]interface{}, len(columns))
	for i := range ptrs {
		ptrs[i] = new(interface{})
	}
	return
}

func rowToMap(rows *sql.Rows, columns []string, ptrs []interface{}) (*core.Obj, error) {
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	item := core.NewObj()
	dest := core.NewMap()
	for i, column := range columns {
		var val string
		v := *(ptrs[i].(*interface{}))
		b, ok := v.([]byte)
		if ok {
			val = string(b)
		} else {
			val = fmt.Sprint(v)
		}
		o := core.NewObj()
		o.Data = val
		dest.SetIndex(column, o)
	}
	item.Data = dest
	return item, nil
}

func SQLQuery(varname string, sqlquery string, pars *core.Array, resvar string) error {
	var (
		db   *sql.DB
		err  error
		rows *sql.Rows
	)
	if db, err = SQLGet(varname); err != nil {
		return ErrInvalidPar
	}
	if rows, err = db.Query(sqlquery, pars.Data...); err != nil {
		return err
	}
	defer rows.Close()
	columns, ptrs, err := sqlColumns(rows)
	if err != nil {
		return err
	}
	obj := core.NewObj()
	list := core.NewArray()

	for rows.Next() {
		item, err := rowToMap(rows, columns, ptrs)
		if err != nil {
			return err
		}
		list.Data = append(list.Data, item)
	}
	obj.Data = list
	if err = rows.Err(); err != nil {
		return err
	}
	SetVarObj(resvar, obj)
	return err
}

func SQLRow(varname string, sqlquery string, pars *core.Array, resvar string) error {
	var (
		db   *sql.DB
		err  error
		rows *sql.Rows
	)
	if db, err = SQLGet(varname); err != nil {
		return ErrInvalidPar
	}
	if rows, err = db.Query(sqlquery, pars.Data...); err != nil {
		return err
	}
	defer rows.Close()
	columns, ptrs, err := sqlColumns(rows)
	if err != nil {
		return err
	}
	if !rows.Next() {
		return sql.ErrNoRows
	}
	obj, err := rowToMap(rows, columns, ptrs)
	if err != nil {
		return err
	}
	if err = rows.Err(); err != nil {
		return err
	}
	SetVarObj(resvar, obj)
	return err
}

func SQLValue(varname string, sqlquery string, pars *core.Array, resvar string) error {
	var (
		db   *sql.DB
		err  error
		rows *sql.Rows
	)
	if db, err = SQLGet(varname); err != nil {
		return ErrInvalidPar
	}
	if rows, err = db.Query(sqlquery, pars.Data...); err != nil {
		return err
	}
	defer rows.Close()
	_, ptrs, err := sqlColumns(rows)
	if err != nil {
		return err
	}
	if !rows.Next() {
		return sql.ErrNoRows
	}
	var ret string
	if err := rows.Scan(ptrs...); err != nil {
		return err
	}
	if len(ptrs) > 0 {
		v := *(ptrs[0].(*interface{}))
		b, ok := v.([]byte)
		if ok {
			ret = string(b)
		} else {
			ret = fmt.Sprint(v)
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	SetVar(resvar, ret)
	return err
}
