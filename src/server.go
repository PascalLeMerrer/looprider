package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"

	"github.com/speps/go-hashids"

	"encoding/json"

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
	Y     int     `json:"y"`
}

// Action represents a player action, like dropping an object on the ground, or triggering an earthquake
type Action struct {
	Type  string  `json:"type,omitempty"`  // the action type
	Extra string  `json:"extra,omitempty"` // extra info about the action; by example the type of object to drop
	Angle float64 `json:"angle"`
	Y     int     `json:"y"`
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
			switch action.Type {
			case "drop":
				createItem(action)
			}

		} else {
			fmt.Println(decodingErr)
		}
	}
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

func sendState(ws *websocket.Conn) {
	jsonBytes, _ := json.Marshal(items)
	sendError := websocket.Message.Send(ws, string(jsonBytes))
	if sendError != nil {
		fmt.Println(sendError)
		return
	}
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
			log.Panicf("Error while converting string to int array %s", err)
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
		ws := c.Socket()
		go receiveActions(ws)
		for {
			animate(speed)
			sendState(ws)
			time.Sleep(time.Millisecond * 100)
		}
	})

	e.Run(":1323")
}
