// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package lib

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type myIDs struct {
	Counter int
}

func (s *myIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	s.Counter++
	return []byte(fmt.Sprintf("id%d", s.Counter))
}

func (s *myIDs) Put(value []byte) {
}

func Markdown(input string) (string, error) {
	ctx := parser.NewContext(parser.WithIDs(&myIDs{}))

	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					html.WithLineNumbers(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	var markDown bytes.Buffer
	if err := markdown.Convert([]byte(input), &markDown, parser.WithContext(ctx)); err != nil {
		return ``, err
	}
	ret := strings.ReplaceAll(markDown.String(), `<code>`, ``)
	return strings.ReplaceAll(ret, `</code>`, ``), nil
}
