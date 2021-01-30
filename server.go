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
	"os"
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
	Port int
	Open bool // if true then webpage is opened
}

type Response struct {
	Success bool `json:"success"`
	//	Message string `json:"message,omitempty"`
	Error string `json:"error,omitempty"`
}

type Auth struct {
	echo.Context
	User *User
	Lang string
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
						if sign == strings.ToLower(.EncodeToString(hash[:])) {
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
	var (
		err error
		url string
	)
	if c.Get(`tpl`) != nil {
		url = c.Get(`tpl`).(string)
	} else {
		url = c.Request().URL.String()
		if url == `/` {
			if IsScript {
				url = `script`
			} else {
				url = `index`
			}
		}
	}
	data, err := RenderPage(c, url)
	if err != nil {
		if err == ErrNotFound {
			err = echo.NewHTTPError(http.StatusNotFound)
		}
		return err
	}
	return c.HTML(http.StatusOK, data)
}

/*func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	message := http.StatusText(code)
	c.HTML(code, message)

}*/

func exitHandle(c echo.Context) error {
	if cfg.playground {
		return jsonError(c, `Access denied`)
	}
	golog.Info(`Shutdown`)
	stopchan <- os.Interrupt
	return c.JSON(http.StatusOK, Response{Success: true})
}

func fileHandle(c echo.Context) error {
	fname := c.Request().URL.String()
	/*	if off := strings.IndexByte(fname, '?'); off > 0 {
		fname = fname[:off]
	}*/
	data := bytes.NewReader(WebAsset(fname))
	http.ServeContent(c.Response(), c.Request(), fname, time.Now(), data)
	return nil //c.HTML(http.StatusOK, Success)
}

func logoutHandle(c echo.Context) error {
	storage.PassCounter++
	if err := SaveStorage(); err != nil {
		return jsonError(c, err)
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}

func reloadHandle(c echo.Context) error {
	ClearAsset()
	InitTemplates()
	InitLang()
	InitScripts()
	return c.JSON(http.StatusOK, Response{Success: true})
}

func installHandle(c echo.Context) error {
	lang := c.FormValue("lang")
	firstRun = false
	if _, ok := langsId[lang]; ok {
		for id, user := range userSettings {
			if user.Lang != lang {
				user.Lang = lang
				userSettings[id] = user
				if err := SaveUser(id); err != nil {
					return jsonError(c, err)
				}
			}
		}
	}
	return c.JSON(http.StatusOK, Response{Success: true})
}

func RunServer(options WebSettings) *echo.Echo {
	InitLang()
	InitTemplates()
	e := echo.New()

	e.HideBanner = true
	e.Use(AuthHandle)
	e.Use(Logger)
	e.Use(md.Recover())
	/* e.Use(md.CORSWithConfig(md.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet},
	}))}*/

	//e.HTTPErrorHandler = customHTTPErrorHandler

	e.GET("/", indexHandle)
	e.GET("/ping", pingHandle)

	e.GET("/js/*", fileHandle)
	e.GET("/css/*", fileHandle)
	e.GET("/images/*", fileHandle)
	e.GET("/webfonts/*", fileHandle)
	e.GET("/favicon.ico", fileHandle)
	if IsScript {
		e.GET("/ws", wsTaskHandle)
		e.GET("/sys", sysHandle)
		e.GET("/info", infoHandle)
		e.POST("/stdin", stdinHandle)
		e.POST("/form", formHandle)
	} else {
		e.GET("/ws", wsMainHandle)
		e.GET("/task/:id", showTaskHandle)
		e.GET("/api/compile", compileHandle)
		e.GET("/api/exit", exitHandle)
		e.GET("/api/export", exportHandle)
		e.GET("/api/reload", reloadHandle)
		e.GET("/api/logout", logoutHandle)
		e.GET("/api/run", runHandle)
		e.GET("/api/script", getScriptHandle)
		e.GET("/api/list", listScriptHandle)
		e.GET("/api/listrun", listRunHandle)
		e.GET("/api/notifications", nfyHandle)
		e.GET("/api/tasks", tasksHandle)
		e.GET("/api/prosettings", proSettingsHandle)
		e.GET("/api/remove/:id", removeTaskHandle)
		e.GET("/api/removenfy/:id", removeNfyHandle)
		e.GET("/api/sys", sysTaskHandle)
		e.GET("/api/settings", settingsHandle)
		e.GET("/api/latest", latestVerHandle)
		e.GET("/api/trial/:id", trialHandle)
		e.POST("/api/install", installHandle)
		e.POST("/api/login", loginHandle)
		e.POST("/api/script", saveScriptHandle)
		e.POST("/api/delete", deleteScriptHandle)
		e.POST("/api/taskstatus", taskStatusHandle)
		e.POST("/api/import", importHandle)
		e.POST("/api/notification", notificationHandle)
		e.POST("/api/settings", saveSettingsHandle)
		e.POST("/api/setpsw", setPasswordHandle)
		e.POST("/api/favs", saveFavsHandle)
	}
	go func() {
		if IsScript {
			e.Logger.SetOutput(ioutil.Discard)
		}
		if err := e.Start(fmt.Sprintf(":%d", options.Port)); err != nil && !isShutdown {
			if IsScript {
				setStatus(TaskFailed, err)
			}
			if cfg.HTTP.Access != AccessHost && pingHost(options.Port) {
				lib.Open(fmt.Sprintf("http://%s:%d", Localhost, options.Port))
			}
			golog.Fatal(err)
		}
	}()
	if options.Open {
		go func() {
			for !pingHost(options.Port) {
				time.Sleep(100 * time.Millisecond)
			}
			lib.Open(fmt.Sprintf("http://%s:%d", Localhost, options.Port))
		}()
	}
	return e
}

func pingHost(port int) bool {
	var (
		body []byte
	)
	resp, err := http.Get(fmt.Sprintf("http://%s:%d/ping", Localhost, port))
	if err == nil {
		body, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	return string(body) == Success
}
