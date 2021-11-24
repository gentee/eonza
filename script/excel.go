// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"strings"

	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
	excel "github.com/xuri/excelize/v2"
)

type Excel struct {
	File *excel.File
}

type ExcelRows struct {
	Rows    *excel.Rows
	Columns []string
}

func OpenXLSX(fname string) (*Excel, error) {
	f, err := excel.OpenFile(fname)
	if err != nil {
		return nil, err
	}
	xls := &Excel{
		File: f,
	}
	return xls, nil
}

/*func XLSXSetSheet(xls *Excel, i int64) {
	xls.File.SetASctiveSheet(i)
}*/

func XLSheetName(xls *Excel, index int64) string {
	return xls.File.GetSheetName(int(index))
}

func XLRows(xls *Excel, sheet, columns string) (*ExcelRows, error) {
	rows, err := xls.File.Rows(sheet)
	if err != nil {
		return nil, err
	}
	var cols []string
	if len(columns) > 0 {
		cols = strings.Split(columns, `,`)
	}
	return &ExcelRows{Rows: rows, Columns: cols}, nil
}

func XLNextRow(rows *ExcelRows) int64 {
	if rows.Rows.Next() {
		return 1
	}
	return 0
}

func XLGetRow(rows *ExcelRows) (*core.Obj, error) {
	cols, err := rows.Rows.Columns()
	if err != nil {
		return nil, err
	}
	if len(rows.Columns) == 0 {
		return vm.IfaceToObj(cols)
	}
	mapcols := make(map[string]interface{})
	for i, col := range rows.Columns {
		var val string
		if i < len(cols) {
			val = cols[i]
		}
		mapcols[col] = val
	}
	return vm.IfaceToObj(mapcols)
}

func XLGetCell(xls *Excel, sheet, axis string) (string, error) {
	return xls.File.GetCellValue(sheet, axis)
}
