package main

import (
	"fmt"
	"log"
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
	ID    string  `json:"id"`
	Kind  string  `json:"kind"` // the object type
	Angle float64 `json:"angle"`
	Y     float64 `json:"y"`
}

// Action represents a player action, like dropping an object on the ground, or triggering an earthquake
type Action struct {
	Type  string  `json:"action"`          // the action type
	Extra string  `json:"extra,omitempty"` // extra info about the action; by example the type of object to drop
	Angle float64 `json:"angle"`
	Y     float64 `json:"y"`
}

// Player represents the user avatar in the game
type Player struct {
	ID              string  `json:"id"`
	InitialPosition float64 `json:"initialPosition"`
	lastPing        time.Time
}

var items []Item
var running bool // is the game in progress?
var players []*Player

// TimeOutMilis is the max time between two reception of a keep alive message from a given client
const TimeOutMilis = 2000

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
		case "join":
			join(action)
		case "start":
			start(ws)
		case "stop":
			stop()
		case "drop":
			createItem(action)
		case "destroy":
			destroyItem(action)
		case "keepAlive":
			keepAlive(action)
		}
	}
	ws.Close()
	return err
}

// stops the current game
func stop() {
	running = false
}

// invoked when a new player joins the game
func join(action *Action) {
	if action.Extra == "" {
		fmt.Println("Missing player nickname in extra parameter of connect action.")
		return
	}
	player := Player{
		action.Extra,
		0,
		time.Now(),
	}
	players = append(players, &player)
	fmt.Printf("Player %s joined the game\n", player.ID)
}

// maintains a given player in the list of active players
// if the player does not invoke keepAlive at least once per second,
// it is kicked from the player list
// it allows detecting the disconnection of clients
// or those which a poor connection
func keepAlive(action *Action) {
	for _, player := range players {
		if player.ID == action.Extra {
			player.lastPing = time.Now()
			fmt.Printf("player %v lastPing updated to %v \n", player.ID, player.lastPing)
		}

	}
	removeDisconnectedPlayers()
}

func removeDisconnectedPlayers() {
	now := time.Now()
	for i, player := range players {
		fmt.Println(now.Sub(player.lastPing) / 1000)
		if now.Sub(player.lastPing) > TimeOutMilis*time.Millisecond {
			fmt.Printf("kicking %s lastPing is %v \n", player.ID, player.lastPing)
			if len(players) > 1 {
				players = append(players[:i], players[i+1:]...)
			} else {
				players = make([]*Player, 0)
			}
		}
	}
}

// removes the disconnected players from the player list
// func removeDisconnected(activePlayers *[]Player) {
// var newPlayers []Player
// for {
// 	now := time.Now()
// 	for i, player := range *activePlayers {
// 		fmt.Printf("player %v lastPing %v \n", player.ID, player.lastPing)
// 		fmt.Printf("now is %v\n", now)
// 		fmt.Printf("diff is  %v \n", now.Sub(player.lastPing))
// 		if now.Sub(player.lastPing) > TimeOutMilis*time.Millisecond {
// 			fmt.Printf("kicking %s\n", player.ID)
// 			if len(*activePlayers) > 1 {
// 				newPlayers = append((*activePlayers)[:i], (*activePlayers)[i+1:]...)
// 				activePlayers = &newPlayers
// 			} else {
// 				newPlayers = make([]Player, 0)
// 				activePlayers = &newPlayers
// 			}
// 		}
// 	}
// 	time.Sleep(time.Millisecond * 500)
// }
// }

// starts a new game
// the inital position of players is computed then sent to client
func start(ws *websocket.Conn) error {
	removeDisconnectedPlayers()
	count := len(players)
	angleBetweenPlayers := 360 / count
	for i, player := range players {
		player.InitialPosition = float64(i * angleBetweenPlayers)
	}

	sendError := websocket.JSON.Send(ws, players)
	if sendError != nil {
		fmt.Printf("Sending Initial State caused an error: %s\n", sendError)
		return sendError
	}

	items = make([]Item, 0, 1000)

	running = true
	return nil
}

// create the model for the item dropped by the player
func createItem(action *Action) {
	item := Item{
		generateID(),
		action.Extra,
		action.Angle,
		action.Y,
	}
	items = append(items, item)
}

// The player wants to destroy an item, remove it from the model
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

func main() {

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

	// go removeDisconnected(&players)

	e.Run(":1323")
}
