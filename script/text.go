// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

func ConvertText(source, from, to string) (string, error) {
	var b bytes.Buffer
	if from != `utf-8` {
		e, err := ianaindex.IANA.Encoding(from)
		if err != nil {
			return ``, err
		}
		toutf8 := transform.NewReader(strings.NewReader(source), e.NewDecoder())
		decBytes, _ := io.ReadAll(toutf8)
		source = string(decBytes)
	}
	if to != `utf-8` {
		e, err := ianaindex.IANA.Encoding(to)
		if err != nil {
			return ``, err
		}
		toutf := transform.NewWriter(&b, e.NewEncoder())
		toutf.Write([]byte(source))
		toutf.Close()
		source = b.String()
	}

	return source, nil
}
