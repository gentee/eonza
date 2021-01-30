// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build !pro

package main

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

const Pro = false

func ProInit() {
}

func SetActive(active bool) error {
	return nil
}

func proSettingsHandle(c echo.Context) error {
	return jsonError(c, fmt.Errorf(`Unsupported`))
}
