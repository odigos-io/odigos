package sse

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

type SSEMessage struct {
	Time string `json:"time"`
	Type string `json:"type"`
	Data string `json:"data"`
}

// This map will hold channels for each client connected to the SSE endpoint
var clients = make(map[chan SSEMessage]bool)

// Function to handle SSE connections
func HandleSSEConnections(c *gin.Context) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Create a new channel for this client
	messageChan := make(chan SSEMessage)
	// Register the channel for this client
	clients[messageChan] = true

	// Remove the channel from the map when this client closes the connection
	defer func() {
		delete(clients, messageChan)
		close(messageChan)
	}()

	// Continuously send SSE messages to the client
	for {
		select {
		case message := <-messageChan:
			// Marshal the message to JSON
			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling JSON: %s", err)
				continue
			}
			// Send the message to the client
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
			c.Writer.Flush()
		case <-c.Writer.CloseNotify():
			// Client connection closed, stop sending messages
			return
		}
	}
}

// // Function to send a message to all connected clients
// func sendMessageToClients(message SSEMessage) {
// 	for client := range clients {
// 		client <- message
// 	}
// }

// Function to send a message to the client
func SendMessageToClient(message SSEMessage) {
	for client := range clients {
		client <- message
		// For single client, you can break after sending the message
		break
	}
}
