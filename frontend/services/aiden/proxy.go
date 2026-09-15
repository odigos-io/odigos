package aiden

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// Next.js `next dev` serves the webapp on a different origin than the Go
	// API (localhost:3000 vs :8085). Production is same-origin.
	CheckOrigin: func(*http.Request) bool { return true },
}

// HandleGatewayWS upgrades the browser connection and pipes it to the Aiden
// OpenClaw gateway, injecting the shared gateway token into the connect frame.
func HandleGatewayWS(c *gin.Context) {
	if !IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "aiden is not enabled"})
		return
	}

	backendURL, err := gatewayWebSocketURL()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "aiden gateway is not configured"})
		return
	}

	client, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("aiden: websocket upgrade: %v", err)
		return
	}
	defer client.Close()

	header := http.Header{}
	if token := gatewayToken(); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}

	backend, _, err := websocket.DefaultDialer.Dial(backendURL.String(), header)
	if err != nil {
		log.Printf("aiden: dial gateway: %v", err)
		_ = client.WriteJSON(map[string]any{"type": "res", "ok": false, "error": "aiden gateway is unreachable"})
		return
	}
	defer backend.Close()

	errc := make(chan struct{}, 2)

	go func() {
		defer func() { errc <- struct{}{} }()
		injected := false
		for {
			messageType, payload, readErr := client.ReadMessage()
			if readErr != nil {
				return
			}
			if !injected && messageType == websocket.TextMessage {
				payload = injectConnectToken(payload, gatewayToken())
				injected = true
			}
			if writeErr := backend.WriteMessage(messageType, payload); writeErr != nil {
				return
			}
		}
	}()

	go func() {
		defer func() { errc <- struct{}{} }()
		for {
			messageType, payload, readErr := backend.ReadMessage()
			if readErr != nil {
				return
			}
			if writeErr := client.WriteMessage(messageType, payload); writeErr != nil {
				return
			}
		}
	}()

	<-errc
}

func gatewayWebSocketURL() (*url.URL, error) {
	raw := strings.TrimSpace(gatewayURL())
	if raw == "" {
		return nil, fmt.Errorf("empty aiden gateway url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return nil, fmt.Errorf("unsupported aiden gateway scheme %q", u.Scheme)
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u, nil
}

// injectConnectToken merges auth.token into the OpenClaw connect frame so the
// browser never needs the shared secret.
func injectConnectToken(raw []byte, token string) []byte {
	if token == "" {
		return raw
	}
	var frame map[string]any
	if err := json.Unmarshal(raw, &frame); err != nil {
		return raw
	}
	if frame["type"] != "req" || frame["method"] != "connect" {
		return raw
	}
	params, _ := frame["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
		frame["params"] = params
	}
	auth, _ := params["auth"].(map[string]any)
	if auth == nil {
		auth = map[string]any{}
		params["auth"] = auth
	}
	auth["token"] = token
	out, err := json.Marshal(frame)
	if err != nil {
		return raw
	}
	return out
}
