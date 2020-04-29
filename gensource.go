// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import "fmt"

type Source struct {
	Linked map[string]bool
	Funcs  string
	Body   string
}

func processScript(script *Script, src *Source) (body string) {

	return
}

func GenSource(script *Script) string {
	src := Source{
		Linked: make(map[string]bool),
		Body: `Println("Hello")
		//ReadString("ok")
		`,
	}
	return fmt.Sprintf("%s\r\nrun {\r\n%s\r\n}", src.Funcs, src.Body)
}
