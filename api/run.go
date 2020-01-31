// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package api

import (
	"eonza/script"
)

func Run(name string) error {
	script.Send()
	return nil
}
