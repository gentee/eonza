// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import "html/template"

type DefItem struct {
	Title string
	Value string
}

type DefList struct {
	Name  template.JS
	Items []DefItem
}

var (
	defaultList = []DefList{
		{
			Name: `regtype`,
			Items: []DefItem{
				{`String (SZ)`, `1`},
				{`Expand String (EXPAND_SZ)`, `2`},
				{`Number (DWORD)`, `4`},
			},
		},
		{
			Name: `regaccess`,
			Items: []DefItem{
				{`---`, `0`},
				{`WOW64_32KEY`, `0x00200`},
				{`WOW64_64KEY`, `0x00100`},
			},
		},
		{
			Name: `regkeys`,
			Items: []DefItem{
				{`HKEY_CLASSES_ROOT`, `0`},
				{`HKEY_CURRENT_USER`, `1`},
				{`HKEY_LOCAL_MACHINE`, `2`},
				{`HKEY_USERS`, `3`},
				{`HKEY_CURRENT_CONFIG`, `4`},
				//				{`HKEY_PERFORMANCE_DATA`, `5`},
			},
		},
		{
			Name: `charmaps`,
			Items: []DefItem{
				{`utf-8`, `utf-8`},
				{`Big5 (Chinese - traditional)`, `Big5`},
				{`cp437  (IBM PC US)`, `cp437`},
				{`cp866  (MS-DOS Cyrillic Russian)`, `cp866`},
				{`EUC-KR (Korean)`, `EUC-KR`},
				{`GBK (Chinese - simplified)`, `GBK`},
				{"KOI8-R", "KOI8-R"},
				{"KOI8-U", "KOI8-U"},
				{`Shift JIS (Japanese)`, `Shift_JIS`},
				{`windows-1250 (Central European)`, `windows-1250`},
				{`windows-1251 (Cyrillic)`, `windows-1251`},
				{`windows-1252 (Western European)`, `windows-1252`},
				{`windows-1253 (Greek)`, `windows-1253`},
				{`windows-1254 (Turkish)`, `windows-1254`},
				{`windows-1255 (Hebrew)`, `windows-1255`},
				{`windows-1256 (Arabic)`, `windows-1256`},
				{`windows-1257 (Baltic)`, `windows-1257`},
				{`windows-1258 (Vietnamese)`, `windows-1258`},
			},
		},
	}
)
