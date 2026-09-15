package aiden

import (
	"encoding/json"
	"testing"
)

func TestInjectConnectToken(t *testing.T) {
	in := []byte(`{"type":"req","id":"1","method":"connect","params":{"role":"operator"}}`)
	out := injectConnectToken(in, "secret-token")

	var frame map[string]any
	if err := json.Unmarshal(out, &frame); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	params := frame["params"].(map[string]any)
	auth := params["auth"].(map[string]any)
	if auth["token"] != "secret-token" {
		t.Fatalf("token=%v", auth["token"])
	}
	if params["role"] != "operator" {
		t.Fatalf("role was dropped: %v", params["role"])
	}
}

func TestInjectConnectTokenIgnoresOtherMethods(t *testing.T) {
	in := []byte(`{"type":"req","id":"2","method":"chat.send","params":{}}`)
	out := injectConnectToken(in, "secret-token")
	if string(out) != string(in) {
		t.Fatalf("expected chat.send to pass through, got %s", out)
	}
}

func TestGatewayWebSocketURL(t *testing.T) {
	t.Setenv("AIDEN_GATEWAY_URL", "http://odigos-aiden.odigos-system.svc:18789")
	t.Setenv("AIDEN_GATEWAY_TOKEN", "tok")
	u, err := gatewayWebSocketURL()
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "ws" || u.Host != "odigos-aiden.odigos-system.svc:18789" {
		t.Fatalf("got %s", u)
	}
}

func TestIsEnabled(t *testing.T) {
	t.Setenv("AIDEN_GATEWAY_URL", "")
	t.Setenv("AIDEN_GATEWAY_TOKEN", "")
	if IsEnabled() {
		t.Fatal("expected disabled")
	}
	t.Setenv("AIDEN_GATEWAY_URL", "http://aiden:18789")
	if IsEnabled() {
		t.Fatal("expected disabled without token")
	}
	t.Setenv("AIDEN_GATEWAY_TOKEN", "tok")
	if !IsEnabled() {
		t.Fatal("expected enabled")
	}
}
