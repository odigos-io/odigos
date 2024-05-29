package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
)

type SSEMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data"`
	Event   string `json:"event"`
	Target  string `json:"target"`
	CRDType string `json:"crdType"`
}

var (
	clients   = make(map[chan SSEMessage]bool)
	clientsMu sync.Mutex
)

func HandleSSEConnections(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	messageChan := make(chan SSEMessage)
	clientsMu.Lock()
	clients[messageChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, messageChan)
		clientsMu.Unlock()
		close(messageChan)
	}()

	for {
		select {
		case message := <-messageChan:
			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling JSON: %s", err)
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
			c.Writer.Flush()
		case <-c.Writer.CloseNotify():
			return
		}
	}
}

func SendMessageToClient(message SSEMessage) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for client := range clients {
		select {
		case client <- message:
		default:
			log.Printf("Channel is closed for client: %v", client)
		}
		break
	}
}
