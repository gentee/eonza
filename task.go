// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

const (
	WcClose = iota // close connection
	WcStatus
)

type WsCmd struct {
	Cmd    int `json:"cmd"`
	Status int `json:"status,omitempty"`
}

func sendStatus(status int, pars ...interface{}) {
	task := TaskStatus{
		TaskID: scriptTask.Header.TaskID,
		Status: status,
	}
	if len(pars) > 0 {
		task.Message = fmt.Sprint(pars[0])
	}
	wsChan <- WsCmd{Cmd: WcStatus, Status: status}
}

var (
	upgrader = websocket.Upgrader{}
	wsChan   = make(chan WsCmd)
	clients  = make(map[*websocket.Conn]bool)
)

func wsTaskHandle(c echo.Context) error {
	//	var cmd WsCmd
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	clients[ws] = true
	/*	defer ws.Close()
		fmt.Println(`Connected`)
		for {
			cmd = <-wsChan
			// Write
			//		err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
			err := ws.WriteJSON(cmd)
			if err != nil {
				// TODO: what's about error?
				fmt.Println(err)
			}
					// Read
					_, msg, err := ws.ReadMessage()
					if err != nil {
						c.Logger().Error(err)
					}
					fmt.Printf("%s\n", msg)
		}*/
	return nil
}
