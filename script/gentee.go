// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentee/gentee"
	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
	"gopkg.in/yaml.v3"
)

type FileLines struct {
	File    *os.File
	Scanner *bufio.Scanner
}

type CSV struct {
	File    *os.File
	Reader  *csv.Reader
	Row     []string
	Columns []string
}

func ObjHandle(handle interface{}) *core.Obj {
	var ret = core.NewObj()
	ret.Data = handle
	return ret
}

func YamlToMap(in string) (*core.Map, error) {
	var (
		tmp map[string]string
		ret interface{}
		err error
	)
	if err = yaml.Unmarshal([]byte(in), &tmp); err != nil {
		return nil, err
	}
	ret, err = gentee.Go2GenteeType(tmp, `map.str`)
	if err != nil {
		return nil, err
	}
	return ret.(*core.Map), nil
}

func CopyName(rt *vm.Runtime, fname string) (string, error) {
	var (
		err error
		i   int
	)
	if !filepath.IsAbs(fname) {
		fname, err = filepath.Abs(fname)
		if err != nil {
			return ``, err
		}
	}
	dir := filepath.Dir(fname)
	base := strings.SplitN(filepath.Base(fname), `.`, 2)
	if len(base) == 1 {
		base = append(base, ``)
	} else {
		base[1] = `.` + base[1]
	}
	for {
		i++
		exist, err := vm.ExistFile(rt, fname)
		if err != nil {
			return ``, err
		}
		if exist == 0 {
			break
		}
		fname = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base[0], i, base[1]))
	}
	return fname, nil
}

func CloseLines(flines *FileLines) error {
	return flines.File.Close()
}

func GetLine(flines *FileLines) string {
	return flines.Scanner.Text()
}

func ReadLines(filename string) (*FileLines, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	return &FileLines{File: file, Scanner: scanner}, nil
}

func ScanLines(flines *FileLines) int64 {
	if flines.Scanner.Scan() {
		return 1
	}
	return 0
}

/*
func ifaceToObj(val interface{}) (*core.Obj, error) {
	ret := core.NewObj()
	switch v := val.(type) {
	case bool:
		ret.Data = v
	case string:
		ret.Data = v
	case int:
		ret.Data = v
	case int64:
		ret.Data = v
	case float64:
		ret.Data = v
	case []string:
		data := core.NewArray()
		data.Data = make([]interface{}, len(v))
		for i, item := range v {
			iobj := core.NewObj()
			iobj.Data = item
			data.Data[i] = iobj
		}
		ret.Data = data
	case []interface{}:
		data := core.NewArray()
		data.Data = make([]interface{}, len(v))
		for i, item := range v {
			iobj, err := ifaceToObj(item)
			if err != nil {
				return nil, err
			}
			data.Data[i] = iobj
		}
		ret.Data = data
	case map[string]interface{}:
		var i int
		data := core.NewMap()
		data.Keys = make([]string, len(v))
		for key, vi := range v {
			data.Keys[i] = key
			iobj, err := ifaceToObj(vi)
			if err != nil {
				return nil, err
			}
			data.Data[key] = iobj
			i++
		}
		ret.Data = data
	case map[interface{}]interface{}:
		var i int
		data := core.NewMap()
		data.Keys = make([]string, len(v))
		for key, vi := range v {
			ikey := fmt.Sprint(key)
			data.Keys[i] = ikey
			iobj, err := ifaceToObj(vi)
			if err != nil {
				return nil, err
			}
			data.Data[ikey] = iobj
			i++
		}
		ret.Data = data
	default:
		return nil, fmt.Errorf("unsupported object type %T", val)
	}
	return ret, nil
}
*/
// YamlToObj converts json to object
func YamlToObj(input string) (ret *core.Obj, err error) {
	var v interface{}
	if err = yaml.Unmarshal([]byte(input), &v); err != nil {
		return
	}
	return vm.IfaceToObj(v)
}

func CloseCSV(hcsv *CSV) error {
	return hcsv.File.Close()
}

func GetCSV(hcsv *CSV) (ret *core.Obj, err error) {
	if len(hcsv.Columns) > 0 {
		ret = core.NewObj()
		data := core.NewMap()
		data.Keys = make([]string, len(hcsv.Columns))
		for i, column := range hcsv.Columns {
			var val string
			if len(column) == 0 {
				continue
			}
			data.Keys[i] = column
			iobj := core.NewObj()
			if i < len(hcsv.Row) {
				val = hcsv.Row[i]
			}
			iobj.Data = val
			data.Data[column] = iobj
		}
		ret.Data = data
		return
	}
	return vm.IfaceToObj(hcsv.Row)
}

func OpenCSV(filename, delim, columns string) (*CSV, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	csvReader := csv.NewReader(file)
	delim = strings.TrimSpace(delim)
	if len(delim) != 0 {
		rdelim := []rune(delim)
		csvReader.Comma = rdelim[0]
	}
	var icolumns []string
	if len(columns) > 0 {
		icolumns = strings.Split(columns, `,`)
	}
	return &CSV{File: file, Reader: csvReader, Columns: icolumns}, nil
}

func ReadCSV(hcsv *CSV) (int64, error) {
	var err error
	hcsv.Row, err = hcsv.Reader.Read()
	if err != nil {
		if err == io.EOF {
			return 0, nil
		}
		return 0, err
	}
	return 1, nil
}

func JSONRequest(urlPath string, jsonData string, headers *core.Map, response string) (ret string, err error) {
	var (
		req *http.Request
		buf []byte
	)
	method := `POST`
	if len(jsonData) == 0 {
		method = `GET`
	}

	if scriptTask.Header.IsPlayground {
		return ``, fmt.Errorf(`Access denied`)
	}
	if req, err = http.NewRequest(method, urlPath, bytes.NewBuffer([]byte(jsonData))); err != nil {
		return
	}
	for _, key := range headers.Keys {
		req.Header.Set(key, headers.Data[key].(string))
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, err := http.DefaultClient.Do(req)
	if err == nil {
		if len(response) > 0 {
			obj, _ := vm.JsonToObj(fmt.Sprintf(`{"statuscode": %d, "status": "%s"}`,
				res.StatusCode, res.Status))
			SetVarObj(response, obj)
		}
		buf, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err == nil {
			ret = string(buf)
		}
	}
	return
}

func TempFile(path, name, content string) (ret string, err error) {
	var f *os.File
	f, err = os.CreateTemp(path, name)
	if err != nil {
		return
	}
	if _, err = f.Write([]byte(content)); err != nil {
		f.Close()
		return
	}
	ret = f.Name()
	return ret, f.Close()
}

func ObjToIface(obj *core.Obj) (ret interface{}) {
	switch v := obj.Data.(type) {
	case int64, float64, string, bool:
		ret = obj.Data
	case *core.Array:
		data := make([]interface{}, len(v.Data))
		for i, item := range v.Data {
			data[i] = ObjToIface(item.(*core.Obj))
		}
		ret = data
	case *core.Map:
		data := make(map[string]interface{})
		for _, key := range v.Keys {
			data[key] = ObjToIface(v.Data[key].(*core.Obj))
		}
		ret = data
	case *core.Obj:
		ret = ObjToIface(v)
	}
	return
}
