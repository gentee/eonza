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
	var response Response

	name := c.QueryParam(`name`)
	if _, ok := scripts[name]; !ok {
		response.Error = Lang(`erropen`, name)
	} else if err := script.Encode(script.Header{
		Name:      name,
		AssetsDir: cfg.AssetsDir,
		HTTP: &lib.HTTPConfig{
			Port:  3235,
			Open:  true,
			Theme: cfg.HTTP.Theme,
		},
	}); err != nil {
		response.Error = fmt.Sprint(err)
	} else {
		response.Success = true
	}
	return c.JSON(http.StatusOK, response)
}

func pingHandle(c echo.Context) error {
	return c.HTML(http.StatusOK, Success)
}
