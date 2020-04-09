// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

import (
	"eonza/lib"
	"eonza/script"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

const (
	WcClose = iota // close connection
	WcStatus
)

type WsClient struct {
	Conn *websocket.Conn
}

type WsCmd struct {
	Cmd    int `json:"cmd"`
	Status int `json:"status,omitempty"`
}

var (
	task     Task
	upgrader websocket.Upgrader
	wsChan   chan WsCmd
	clients  map[uint32]WsClient

	console *os.File

	chStdin  chan []byte
	chStdout chan []byte
	chSystem chan int
	chFinish chan bool
)

func initTask() script.Settings {
	task = Task{
		ID:        scriptTask.Header.TaskID,
		UserID:    scriptTask.Header.UserID,
		Status:    TaskActive,
		Name:      scriptTask.Header.Name,
		StartTime: time.Now(),
	}

	console = os.Stdout
	upgrader = websocket.Upgrader{}
	wsChan = make(chan WsCmd)
	clients = make(map[uint32]WsClient)

	chStdin = make(chan []byte)
	chStdout = make(chan []byte)
	chSystem = make(chan int)
	chFinish = make(chan bool)

	go func() {
		var out []byte
		for {
			out = <-chStdout
			console.Write(out)
		}
	}()

	go func() {
		var cmd WsCmd
		for task.Status <= TaskSuspended {
			cmd = <-wsChan
			debug(`get cmd`)
			mutex.Lock()
			debug(`clients`, len(clients))
			for id, client := range clients {
				err := client.Conn.WriteJSON(cmd)
				debug(`ws`, id, err)
				if err != nil {
					client.Conn.Close()
					delete(clients, id)
				}
			}
			mutex.Unlock()
		}
		debug(`disconnect`)
		for _, client := range clients {
			client.Conn.Close()
		}
		chFinish <- true
	}()

	return script.Settings{
		ChStdin:  chStdin,
		ChStdout: chStdout,
		ChSystem: chSystem,
	}
}

func wsTaskHandle(c echo.Context) error {
	//	var cmd WsCmd
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	if err = ws.WriteJSON(WsCmd{
		Cmd:    WcStatus,
		Status: task.Status,
	}); err == nil {
		clients[lib.RndNum()] = WsClient{
			Conn: ws,
		}
	}
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

func setStatus(status int, pars ...interface{}) {
	/*	task := TaskStatus{
			TaskID: scriptTask.Header.TaskID,
			Status: status,
		}
		if len(pars) > 0 {
			task.Message = fmt.Sprint(pars[0])
		}*/
	debug(`cmd`, status)
	wsChan <- WsCmd{Cmd: WcStatus, Status: status}
	debug(`cmd ok`)
	task.Status = status
}

func debug(pars ...interface{}) {
	console.Write([]byte(fmt.Sprintln(pars...)))
}
