package sse

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
)

type MessageType string

const (
	MessageTypeWarning MessageType = "warning"
	MessageTypeError   MessageType = "error"
	MessageTypeSuccess MessageType = "success"
	MessageTypeInfo    MessageType = "info"
	MessageTypeDefault MessageType = "default"
)

type MessageEvent string

const (
	MessageEventAdded    MessageEvent = "Added"
	MessageEventDeleted  MessageEvent = "Deleted"
	MessageEventModified MessageEvent = "Modified"
)

type SSEMessage struct {
	Type    MessageType  `json:"type"`
	Event   MessageEvent `json:"event"`
	Data    string       `json:"data"`
	Target  string       `json:"target"`
	CRDType string       `json:"crdType"`
}

// This map will hold channels for each client connected to the SSE endpoint
var (
	clients   = make(map[chan SSEMessage]bool)
	clientsMu sync.Mutex
)

// Function to handle SSE connections
func HandleSSEConnections(c *gin.Context) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Create a new channel for this client
	messageChan := make(chan SSEMessage, 10)
	// Register the channel for this client
	clientsMu.Lock()
	clients[messageChan] = true
	clientsMu.Unlock()

	// Remove the channel from the map when this client closes the connection
	defer func() {
		clientsMu.Lock()
		delete(clients, messageChan)
		clientsMu.Unlock()
		close(messageChan)
	}()

	// Continuously send SSE messages to the client
	for {
		select {
		case message := <-messageChan:
			jsonData, err := json.Marshal(message)
			if err != nil {
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
			c.Writer.Flush()
		case <-c.Writer.CloseNotify():
			return
		}
	}
}

// Function to send a message to all clients
func SendMessageToClient(message SSEMessage) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for client := range clients {
		select {
		case client <- message:
		default:

		}
	}
}
