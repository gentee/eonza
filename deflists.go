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
			Name: `charmaps`,
			Items: []DefItem{
				{`utf-8`, `utf-8`},
				{`windows-1251`, `windows-1251`},
			},
		},
	}
)
