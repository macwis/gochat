package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type ChatMessage struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

var clients = make(map[*websocket.Conn]bool)
var broadcaster = make(chan ChatMessage)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleConnections(w http.ResponseWriter, r *http.Request) {

	redisUrl := os.Getenv("REDIS_URL")

	rdb, err := redis.DialURL(redisUrl)
	if err != nil {
		panic(err)
	}
	defer rdb.Close()

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// ensure connection close when function returns
	defer ws.Close()
	clients[ws] = true

	// if it's zero, no messages were ever sent/saved
	if exists, err := redis.Bool(rdb.Do("EXISTS", "chat_messages")); err != nil {
		panic(err)
	} else if exists {
		// new connection, sending history
		sendPreviousMessages(rdb, ws)
	}

	for {
		var msg ChatMessage
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			delete(clients, ws)
			break
		}
		// send new message to the channel
		broadcaster <- msg
	}
}

func sendPreviousMessages(rdb redis.Conn, ws *websocket.Conn) {
	messageClient(ws, ChatMessage{Username: "bot", Text: "hello"})

	values, err := redis.Values(rdb.Do("LRANGE", "chat_messages", 0, -1))
	if err != nil {
		panic(err)
	}

	// send previous messages
	for len(values) > 0 {
		var content string
		values, err = redis.Scan(values, &content)

		var msg ChatMessage
		json.Unmarshal([]byte(content), &msg)
		messageClient(ws, msg)
	}
}

// If a message is sent while a client is closing, ignore the error
func unsafeError(err error) bool {
	return !websocket.IsCloseError(err, websocket.CloseGoingAway) && err != io.EOF
}

func handleMessages(rdb redis.Conn) {

	for {
		// grab any next message from channel
		msg := <-broadcaster

		storeInRedis(rdb, msg)
		messageClients(msg)
	}
}

func storeInRedis(rdb redis.Conn, msg ChatMessage) {
	json, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	if _, err := rdb.Do("RPUSH", "chat_messages", json); err != nil {
		panic(err)
	}
}

func messageClients(msg ChatMessage) {
	// send to every client currently connected
	for client := range clients {
		messageClient(client, msg)
	}
}

func messageClient(client *websocket.Conn, msg ChatMessage) {
	err := client.WriteJSON(msg)
	if err != nil && unsafeError(err) {
		log.Printf("error: %v", err)
		client.Close()
		delete(clients, client)
	}
}

func main() {
	redisUrl := os.Getenv("REDIS_URL")

	rdb, err := redis.DialURL(redisUrl)
	if err != nil {
		panic(err)
	}
	defer rdb.Close()

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	port := os.Getenv("PORT")

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/websocket", handleConnections)
	go handleMessages(rdb)

	log.Print("Server starting at localhost:", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}

}
