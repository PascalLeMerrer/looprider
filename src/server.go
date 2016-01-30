package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"

	"github.com/speps/go-hashids"

	"time"

	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"golang.org/x/net/websocket"
)

// Item is an object that can be dropped on the planet ground
type Item struct {
	ID    string  `json:"id,omitempty"`
	Kind  string  `json:"kind,omitempty"` // the object type
	Angle float64 `json:"angle"`
	Y     float64 `json:"y"`
}

// Action represents a player action, like dropping an object on the ground, or triggering an earthquake
type Action struct {
	Type  string  `json:"action,omitempty"` // the action type
	Extra string  `json:"extra,omitempty"`  // extra info about the action; by example the type of object to drop
	Angle float64 `json:"angle"`
	Y     float64 `json:"y"`
}

var items []Item
var running bool // is the game in progress?

func receiveActions(ws *websocket.Conn) error {
	var err error
	for {

		action := new(Action)
		err = websocket.JSON.Receive(ws, &action)
		if err != nil {
			fmt.Printf("Receive error: %s\n", err)
			break
		}

		fmt.Printf("Received %+v\n", action)

		switch action.Type {
		case "start":
			start()
		case "stop":
			stop()
		case "drop":
			createItem(action)
		case "destroy":
			destroyItem(action)
		}

	}
	ws.Close()
	return err
}

func stop() {
	running = false
}

func start() {
	items = make([]Item, 0, 1000)
	running = true
}

func createItem(action *Action) {
	item := Item{
		generateID(),
		action.Extra,
		action.Angle,
		action.Y,
	}
	items = append(items, item)
}

func destroyItem(action *Action) {
	for index, item := range items {
		if item.ID == action.Extra {
			items = append(items[:index], items[index+1:]...)
		}
	}
}

func sendState(ws *websocket.Conn) error {
	sendError := websocket.JSON.Send(ws, items)
	if sendError != nil {
		fmt.Printf("SendState error: %s\n", sendError)
		return sendError
	}
	return nil
}

// returns an 8 characters random string
// the characters may be any of [A-Z][a-z][0-9]
func generateID() string {
	hashIDConfig := hashids.NewData()
	hashIDConfig.Salt = "zs4e6f80KDla1-2xcCD!34%<?23POsd"
	hashIDConfig.MinLength = 8
	hashIDConfig.Alphabet = hashids.DefaultAlphabet
	hash := hashids.NewWithData(hashIDConfig)

	randomInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int63()
	intArray := intToIntArray(randomInt, 8)
	result, _ := hash.Encode(intArray)

	return result
}

// converts an int64 number to a fixed length array of int
func intToIntArray(value int64, length int) []int {
	result := make([]int, length)
	valueAsString := strconv.FormatInt(value, 10)

	fragmentLength := len(valueAsString) / length

	var startIndex, endIndex int
	var intValue int64
	var err error

	for index := 0; index < length; index++ {

		startIndex = index * fragmentLength
		endIndex = ((index + 1) * fragmentLength)

		if endIndex <= len(valueAsString) {
			intValue, err = strconv.ParseInt(valueAsString[startIndex:endIndex], 10, 0)
		} else {
			intValue, err = strconv.ParseInt(valueAsString[startIndex:], 10, 0)
		}

		if err != nil {
			log.Panicf("Error while converting string to int array %s\n", err)
		}
		result[index] = int(intValue)
	}
	return result
}

// update the position of items according to angular speed
func animate(speed float64) {
	count := len(items)
	for i := 0; i < count; i++ {
		items[i].Angle = math.Mod(items[i].Angle+speed, 360)
	}
}

func main() {
	speed := 1.44

	e := echo.New()

	e.Use(mw.Logger())
	e.Use(mw.Recover())

	e.Static("/", "public")
	e.WebSocket("/ws", func(c *echo.Context) (err error) {
		fmt.Println("Creating a new websocket")
		ws := c.Socket()

		go receiveActions(ws)
		for {
			if running {
				animate(speed)
				err := sendState(ws)
				if err != nil {
					break
				}
				time.Sleep(time.Millisecond * 100)
			}
		}
		fmt.Println("Exited loop")
		return err
	})

	e.Run(":1323")
}
