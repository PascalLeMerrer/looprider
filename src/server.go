package main

import (
	"fmt"

	"encoding/json"

	"time"

	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"golang.org/x/net/websocket"
)

// Item is an object that can be dropped on the planet ground
type Item struct {
	ID int `json:"id,omitempty"`
	X  int `json:"x,omitempty"`
	Y  int `json:"y,omitempty"`
}

// Action represents a player action, like dropping an object on the ground, or triggering an earthquake
type Action struct {
	ID int `json:"id,omitempty"`
	X  int `json:"x,omitempty"`
	Y  int `json:"y,omitempty"`
}

var items = make([]Item, 0, 1000)

func receiveActions(ws *websocket.Conn) {
	msg := ""

	for {

		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println(msg)
		action := new(Action)
		decodingErr := json.Unmarshal([]byte(msg), &action)
		if decodingErr == nil {

			item := Item{
				action.ID,
				action.X,
				action.Y,
			}

			items = append(items, item)

		} else {
			fmt.Println(decodingErr)
		}
	}
}

func sendState(ws *websocket.Conn) {
	jsonBytes, _ := json.Marshal(items)
	sendError := websocket.Message.Send(ws, string(jsonBytes))
	if sendError != nil {
		fmt.Println(sendError)
		return
	}
}

func main() {
	// speed := 0

	e := echo.New()

	e.Use(mw.Logger())
	e.Use(mw.Recover())

	e.Static("/", "public")
	e.WebSocket("/ws", func(c *echo.Context) (err error) {
		ws := c.Socket()
		go receiveActions(ws)
		for {
			sendState(ws)
			time.Sleep(time.Millisecond * 100)
		}
	})

	e.Run(":1323")
}
