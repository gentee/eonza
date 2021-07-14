// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gentee/gentee/core"
)

func ParseHTML(input string) (*goquery.Selection, error) {
	var err error

	if !strings.HasPrefix(input, `<`) && IsVar(input) != 0 {
		if input, err = GetVar(input); err != nil {
			return nil, err
		}
	}
	if strings.HasPrefix(strings.TrimSpace(input), `<`) {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
		return doc.Selection, err
	}
	var (
		o   *core.Obj
		ret *goquery.Selection
		ok  bool
	)
	if o, err = GetVarObj(input); err != nil {
		return nil, err
	}
	if ret, ok = o.Data.(*goquery.Selection); !ok {
		return nil, fmt.Errorf(`%s is not html node`, input)
	}
	return ret, nil
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
