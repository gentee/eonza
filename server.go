// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"eonza/lib"

	"github.com/kataras/golog"
	"github.com/labstack/echo/v4"
	md "github.com/labstack/echo/v4/middleware"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
)

// WebSettings contains web-server parameters
type WebSettings struct {
	Domain string // Domain, localhost if it sempty
	Port   int
	Open   bool // if true then webpage is opened
}

var (
	ErrNotFound = errors.New(`Not found`)
	IsScript    bool // true, if web-server for the script
)

func Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			golog.Warn(err)
		}
		return err
		/*		var (
					code  int
					err   error
					msg   string
					valid bool
				)
				req := c.Request()
				if req.URL.String() == `/` {
					return next(c)
				}
				remoteAddr := req.RemoteAddr
				if ip := req.Header.Get(XRealIP); len(ip) > 6 {
					remoteAddr = ip
				} else if ip = req.Header.Get(XForwardedFor); len(ip) > 6 {
					remoteAddr = ip
				}
				if strings.Contains(remoteAddr, ":") {
					remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
				}
				sign := strings.ToLower(c.QueryParam(`hash`))
				forHash := cfg.Password
				device := c.QueryParam(`device`)
				key := c.QueryParam(`key`)
				if len(cfg.Devices) > 0 {
					for _, device := range cfg.Devices {
						hash := md5.Sum([]byte(forHash + device + key))
						if sign == strings.ToLower(hex.EncodeToString(hash[:])) {
							valid = true
							break
						}
					}
				} else {
					hash := md5.Sum([]byte(forHash + key))
					valid = sign == strings.ToLower(hex.EncodeToString(hash[:]))
				}
				if len(device) > 0 && valid {
					err = next(c)
					if err != nil {
						code = http.StatusInternalServerError
						if he, ok := err.(*echo.HTTPError); ok {
							code = he.Code
						}
						msg = http.StatusText(code)
					} else {
						code = c.Response().Status
					}
				} else {
					code = http.StatusUnauthorized
					msg = http.StatusText(code)
				}
				if len(msg) > 0 {
					c.JSON(code, Result{Message: msg})
				}
				url := req.URL.String()
				if ind := strings.IndexByte(url, '?'); ind >= 0 {
					url = url[:ind]
				}
				out := fmt.Sprintf("%s,%s,%s,%d", url, remoteAddr, device, code)
				cmd := c.Get("cmd")
				if cmd != nil {
					out += `,` + cmd.(string)
				}
				isError := c.Get("error")
				if code != http.StatusOK || (isError != nil && isError.(bool)) {
					golog.Warn(out)
				} else {
					golog.Info(out)
				}
				return err*/
	}
}

func indexHandle(c echo.Context) error {
	var err error
	//	req := c.Request()
	url := c.Request().URL.String()
	if url == `/` {
		if IsScript {
			url = `script`
		} else {
			url = `index`
		}
	}
	data, err := RenderPage(url)
	if err != nil {
		if err == ErrNotFound {
			err = echo.NewHTTPError(http.StatusNotFound)
		}
		return err
	}
	return c.HTML(http.StatusOK, data)
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	//	url := fmt.Sprintf("/%d.html", code)
	message := http.StatusText(code)
	/*	if _, ok := pages[url]; ok {
		if data, err := RenderPage(url); err == nil {
			message = data
		}
	}*/
	c.HTML(code, message)
	/*	if err := c.File(errorPage); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)*/
}

func fileHandle(c echo.Context) error {
	fname := c.Request().URL.String()
	/*	if off := strings.IndexByte(fname, '?'); off > 0 {
		fname = fname[:off]
	}*/
	data := bytes.NewReader(FileAsset(fname))
	http.ServeContent(c.Response(), c.Request(), fname, time.Now(), data)
	return nil //c.HTML(http.StatusOK, Success)
}

func RunServer(options WebSettings) {
	chStart := make(chan bool)
	if len(options.Domain) == 0 {
		options.Domain = `localhost`
	}
	e := echo.New()

	e.HideBanner = true
	e.Use(Logger)
	e.Use(md.Recover())

	e.HTTPErrorHandler = customHTTPErrorHandler

	e.GET("/", indexHandle)
	e.GET("/api/ping", pingHandle)
	e.GET("/js/*", fileHandle)
	e.GET("/css/*", fileHandle)
	if !IsScript {
		e.GET("/api/run", runHandle)
	}
	url := fmt.Sprintf("http://%s:%d", options.Domain, options.Port)
	if options.Open {
		go func() {
			<-chStart
			lib.Open(url)
		}()
	}
	go func() {
		var body []byte
		for string(body) != Success {
			resp, err := http.Get(url + `/api/ping`)
			if err == nil {
				body, _ = ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
			time.Sleep(100 * time.Millisecond)
		}
		chStart <- true
	}()
	start := func() {
		if err := e.Start(fmt.Sprintf(":%d", options.Port)); err != nil {
			golog.Fatal(err)
		}
	}
	if IsScript {
		go start()
	} else {
		start()
	}
}
