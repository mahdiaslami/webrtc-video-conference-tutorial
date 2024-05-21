package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	broadcasters       = make(map[string]string)
	connections        = make(map[string]*websocket.Conn)
	connectionSequence = 1

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	router := gin.Default()
	router.Static("/public", "./publicforgo")
	router.GET("/ws", handleConnections)

	log.Println("Listening on port", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(c *gin.Context) {
	log.Println("[WS]: request received")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error during connection upgrade:", err)
		return
	}
	defer conn.Close()

	connId := strconv.Itoa(connectionSequence)
	connectionSequence++

	connections[connId] = conn

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// Handle the incoming messages (e.g., register as broadcaster/viewer, offer, answer, candidate)
		if err := handleMessage(connId, msg); err != nil {
			log.Println("[HM]: ", err)
			return
		}
	}
}

func handleMessage(id string, msg []byte) error {
	// Parse the message and handle it accordingly
	// For the sake of brevity, this is a placeholder function.
	// You need to implement the actual logic for handling messages based on your requirements.
	log.Println("Message received:", string(msg))

	data := make([]any, 0)

	if err := json.Unmarshal(msg, &data); err != nil {
		return err
	}

	eventName := data[0]

	switch eventName {
	case "register as broadcaster":
		room := data[1].(string)
		log.Println("[HM]: register as broadcaster for room", room)
		broadcasters[room] = id

	case "register as viewer":
		user := data[1].(map[string]any)
		log.Println("register as viewer for room", user["room"])

		user["id"] = id

		room := user["room"].(string)
		bconn := connections[broadcasters[room]]
		bconn.WriteJSON(map[string]any{
			"eventName": "new viewer",
			"arguments": []any{user},
		})
	case "candidate":
		targetId := data[1].(string)
		connections[targetId].WriteJSON(map[string]any{
			"eventName": "candidate",
			"arguments": []any{id, data[2]},
		})

	case "offer":
		targetId := data[1].(string)

		ev := data[2].(map[string]any)
		b := ev["broadcaster"].(map[string]any)
		b["id"] = id

		connections[targetId].WriteJSON(map[string]any{
			"eventName": "offer",
			"arguments": []any{b, ev["sdp"]},
		})
	case "answer":
		ev := data[1].(map[string]any)

		room := ev["room"].(string)

		connections[broadcasters[room]].WriteJSON(map[string]any{
			"eventName": "answer",
			"arguments": []any{id, ev["sdp"]},
		})
	default:
		return errors.New("event doesn't exists")
	}

	return nil
}
