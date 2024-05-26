package utils

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Socket struct {
	conn      *websocket.Conn
	listeners map[string]func([]any)
	mu        sync.Mutex
}

func NewSocket() *Socket {
	var err error
	var s = &Socket{
		listeners: map[string]func([]any){},
	}

	u := "ws://localhost:3000/ws"
	log.Printf("[SO]: connecting to %s", u)

	s.conn, _, err = websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		log.Fatal("[SO] dial: ", err)
	}

	go func() {
		defer s.conn.Close()
		for {
			_, msg, err := s.conn.ReadMessage()
			if err != nil {
				log.Println("[SO]: ", err)
				break
			}

			s.handleMessage(msg)
		}
	}()

	return s
}

func (s *Socket) handleMessage(msg []byte) {
	log.Println("[SO] message from server: ", string(msg))

	data := make(map[string]any, 0)
	json.Unmarshal(msg, &data)

	arguments, ok := data["arguments"].([]any)
	if !ok {
		log.Println("[SO] casting arguments failed: ", data["arguments"])
		return
	}

	eventName, ok := data["eventName"].(string)
	if !ok {
		log.Println("[SO] casting eventName failed: ", data["eventName"])
		return
	}
	callback := s.listeners[eventName]

	callback(arguments)
}

func (s *Socket) Emit(a ...any) {
	s.mu.Lock()
	s.conn.WriteJSON(a)
	s.mu.Unlock()
}

func (s *Socket) On(ev string, callback func([]any)) {
	s.listeners[ev] = callback
}
