// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	Counter  int64
	Username string
	jwt.StandardClaims
}

type session struct {
	Token   string
	Created time.Time
}

type ResponseLogin struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error,omitempty"`
}

var (
	sessionKey string
	sessions   = make(map[string]session)
)

func getCookie(c echo.Context, name string) string {
	cookie, err := c.Cookie(name)
	if err != nil {
		return ``
		/*		if err != http.ErrNoCookie {
				return err
			}*/
	}
	return cookie.Value
}

func accessIP(curIP, originalIP string) bool {
	return curIP == originalIP || net.ParseIP(curIP).IsLoopback()
}

func AuthHandle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		var (
			access   string
			isAccess bool
		)
		ip := c.RealIP()
		if len(cfg.Whitelist) > 0 {
			var matched bool
			clientip := net.ParseIP(ip)
			for _, item := range cfg.Whitelist {
				_, network, err := net.ParseCIDR(item)
				if err == nil && network.Contains(clientip) {
					matched = true
					break
				}
			}
			if !matched {
				return echo.NewHTTPError(http.StatusForbidden, "Access denied")
			}
		}
		host := c.Request().Host
		if offPort := strings.LastIndex(c.Request().Host, `:`); offPort > 0 {
			host = host[:offPort]
		}
		if IsScript {
			access = scriptTask.Header.HTTP.Access
		} else {
			access = cfg.HTTP.Access
		}
		if access == AccessPrivate {
			isAccess = lib.IsPrivate(host, ip)
		} else if access == AccessHost {
			if IsScript {
				isAccess = (host == scriptTask.Header.HTTP.Host && accessIP(ip, scriptTask.Header.IP)) ||
					host == Localhost
			} else {
				isAccess = host == cfg.HTTP.Host || host == Localhost
			}
		} else {
			isAccess = lib.IsLocalhost(host, ip)
		}
		if !isAccess {
			return echo.NewHTTPError(http.StatusForbidden, "Access denied")
		}
		mutex.Lock()
		defer mutex.Unlock()

		url := c.Request().URL.String()
		if len(storage.Settings.PasswordHash) > 0 && (url == `/` || strings.HasPrefix(url, `/api`) ||
			strings.HasPrefix(url, `/ws`) || strings.HasPrefix(url, `/task`)) {
			hashid := getCookie(c, "hashid")
			jwtData := getCookie(c, "jwt")
			if len(hashid) > 0 {
				if item, ok := sessions[hashid]; ok {
					c.SetCookie(&http.Cookie{
						Name:     "jwt",
						Value:    item.Token,
						Expires:  time.Now().Add(30 * 24 * time.Hour),
						HttpOnly: true,
					})
					jwtData = item.Token
					delete(sessions, hashid)
				}
				c.SetCookie(&http.Cookie{
					Name:    "hashid",
					Value:   "",
					Path:    "/",
					Expires: time.Unix(0, 0),
				})
			}
			var valid bool
			if len(jwtData) > 0 {
				claims := &Claims{}
				token, err := jwt.ParseWithClaims(jwtData, claims,
					func(token *jwt.Token) (interface{}, error) {
						/*	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
							return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
						*/
						return []byte(cfg.HTTP.JWTKey + sessionKey), nil
					})
				if err == nil {
					if claims.Counter == storage.PassCounter {
						valid = token.Valid
					}
				}
			}
			if !valid {
				if url == `/` {
					c.Request().URL.Path = `login`
				} else if url != `/api/login` && url != `/api/taskstatus` && url != `/api/sys` {
					return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
				}
			}
		}
		// TODO: JWT user
		var user *User
		for _, user = range storage.Users {
			break
		}
		lang := LangDefCode
		if IsScript {
			lang = scriptTask.Header.Lang
		} else {
			if u, ok := userSettings[user.ID]; ok {
				lang = u.Lang
			}
		}
		auth := &Auth{
			Context: c,
			User:    user,
			Lang:    lang,
		}
		err = next(auth)
		return
	}
}

func clearSessions() {
	for id, item := range sessions {
		if time.Since(item.Created).Seconds() > 5.0 {
			delete(sessions, id)
		}
	}
}

func loginHandle(c echo.Context) error {
	var response ResponseLogin

	err := bcrypt.CompareHashAndPassword(storage.Settings.PasswordHash, []byte(c.FormValue("password")))
	if err == nil {
		expirationTime := time.Now().Add(30 * 24 * time.Hour)
		claims := &Claims{
			Counter:  storage.PassCounter,
			Username: `root`,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			},
		}
		var token string
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err = tok.SignedString([]byte(cfg.HTTP.JWTKey + sessionKey))
		if err == nil {
			response.ID = lib.UniqueName(12)
			clearSessions()
			sessions[response.ID] = session{
				Token:   token,
				Created: time.Now(),
			}
		}
	}
	if err != nil {
		response.Error = err.Error()
	}
	response.Success = err == nil
	return c.JSON(http.StatusOK, response)
}
