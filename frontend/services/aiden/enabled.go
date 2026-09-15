// Package aiden reverse-proxies the OpenClaw gateway WebSocket used by the
// in-UI Aiden chat. The browser talks to /api/aiden/ws on the Odigos UI; this
// package dials the in-cluster Aiden Service and injects the gateway token so
// it never leaves the cluster.
package aiden

import (
	"os"
)

const (
	gatewayURLEnv   = "AIDEN_GATEWAY_URL"
	gatewayTokenEnv = "AIDEN_GATEWAY_TOKEN"
)

// IsEnabled reports whether the UI should expose the Aiden chat. Helm sets
// AIDEN_GATEWAY_URL and AIDEN_GATEWAY_TOKEN on the UI pod when aiden.enabled
// is true.
func IsEnabled() bool {
	return os.Getenv(gatewayURLEnv) != "" && os.Getenv(gatewayTokenEnv) != ""
}

func gatewayURL() string {
	return os.Getenv(gatewayURLEnv)
}

func gatewayToken() string {
	return os.Getenv(gatewayTokenEnv)
}
