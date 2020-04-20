// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import "github.com/labstack/echo/v4"

func showTaskHandle(c echo.Context) error {
	return jsonSuccess(c)
}
