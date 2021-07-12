// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gentee/gentee/core"
)

func ParseHTML(input string) (*goquery.Selection, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	return doc.Selection, err
}

func FindHTML(node *goquery.Selection, selector string) *goquery.Selection {
	return node.Find(selector)
}

func AttribHTML(node *goquery.Selection, attrib string) string {
	return node.AttrOr(attrib, "")
}

func TextHTML(node *goquery.Selection) string {
	return node.Text()
}

func ChildrenHTML(node *goquery.Selection) *core.Array {
	ret := core.NewArray()
	ret.Data = make([]interface{}, 0)
	node.Each(func(index int, item *goquery.Selection) {
		ret.Data = append(ret.Data, item)
	})
	return ret
}
