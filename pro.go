// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build pro

package main

import (
	"net/http"

	pro "github.com/gentee/eonza-pro"
	"github.com/labstack/echo/v4"
)

type ProOptions struct {
	Active   bool         `json:"active"`
	Settings pro.Settings `json:"settings"`
	Trial    Trial        `json:"trial"`
}

const (
	Pro = true
)

func SetActive(active bool) error {
	return pro.SetActive(active)
}

func ProInit() {
	pro.LoadPro(storage.Trial.Mode > 0)
}

func proSettingsHandle(c echo.Context) error {
	var response ProOptions

	response.Active = pro.Active
	response.Trial = storage.Trial
	return c.JSON(http.StatusOK, &response)
}
