// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"

	"eonza/lib"
	"eonza/script"

	"github.com/labstack/echo/v4"
)

func runHandle(c echo.Context) error {
	if err := script.Encode(script.Header{
		Name:      "World",
		AssetsDir: cfg.AssetsDir,
		HTTP: &lib.HTTPConfig{
			Port:  3235,
			Open:  true,
			Theme: cfg.HTTP.Theme,
		},
	}); err != nil {
		return err
	}
	return c.HTML(http.StatusOK, "OK")
}

func pingHandle(c echo.Context) error {
	fmt.Println(`PING`)
	return c.HTML(http.StatusOK, Success)
}
