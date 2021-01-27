// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build pro

package main

import (
	pro "github.com/gentee/eonza-pro"
)

const Pro = true

func ProInit() {
	pro.LoadPro()
}
