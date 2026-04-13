package main

import (
	"net/http"
	"sync"
	"time"
	"github.com/gorilla/websocket"
)

type RequestEvent struct {
	Timestamp 	int64	`json:"timestamp"`
	Upstream	string	`json:"upstream"`
	Method 		string 	`json:"method"`
	Path 		string 	`json:"path"`
	Status		string 	`json:"status"`
	Latency		int64	`json:"latency"`
	CircuitOpen	bool	`json:"ciruitopen"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Connected clients
	clients =  make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex

	// Channel to broadcast events
	eventsChan = make(chan RequestEvent, 100)
)

// init runs the Websocket broadcaster
func init() {
	go broadcastEvents()
}

// broadcastEvents listens for events and sends them to all connected clients
func broadcastEvents() {
	for event := range eventsChan {
		clientsMu.Lock()
		for client := range clients {
			err := client.WriteJSON(event)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		clientsMu.Unlock()
	}
}

// handleWebSocket handles new WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()


	// Keep connection open
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			conn.Close()
			break
		}
	}
}

// BroadcastRequest sends a request event to all connected clients
func BroadcastRequest(upstream, method, path, status string, latency int64, circuitOpen bool) {
	event := RequestEvent{
		Timestamp:   time.Now().Unix(),
		Upstream:    upstream,
		Method:      method,
		Path:        path,
		Status:      status,
		Latency:     latency,
		CircuitOpen: circuitOpen,
	}

	select {
	case eventsChan <- event:
	default:
		// Channel full, drop event
	}
}