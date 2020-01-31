// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import "github.com/gentee/gentee"

var (
	workspace *gentee.Gentee
)

func InitWorkspace() {
	workspace = gentee.New()
}

func Compile() {

}
