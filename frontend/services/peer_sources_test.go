package services

import (
	"testing"
	"time"

	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
)

func TestServiceGraphNodeAttributesForServerAndClient(t *testing.T) {
	full := map[string]string{
		"client":                    "user",
		"client_service_name":       "user",
		"client_k8s_namespace_name": "shop",
		"server":                    "redis",
		"server_service_name":       "redis",
		"server_db_system":          "redis",
		"server_net_peer_name":      "redis",
	}

	t.Run("server", func(t *testing.T) {
		got := ServiceGraphNodeAttributesForServer(full)
		if len(got) != 2 {
			t.Fatalf("want 2 keys (bare server and service_name omitted), got %d: %v", len(got), got)
		}
		if got["server"] != "" || got["server_db_system"] != "" {
			t.Error("keys must be unprefixed (no server_ prefix)")
		}
		if got["client"] != "" || got["k8s.namespace.name"] != "" {
			t.Error("client keys must be stripped")
		}
		if got["db.system"] != "redis" || got["net.peer.name"] != "redis" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("client", func(t *testing.T) {
		got := ServiceGraphNodeAttributesForClient(full)
		if len(got) != 1 {
			t.Fatalf("want 1 key (service_name omitted), got %d: %v", len(got), got)
		}
		if got["server_db_system"] != "" || got["db.system"] != "" {
			t.Error("server keys must be stripped")
		}
		if got["client"] != "" || got["service.name"] != "" {
			t.Error("bare client and redundant service.name must be omitted")
		}
		if got["k8s.namespace.name"] != "shop" {
			t.Errorf("k8s.namespace.name = %q", got["k8s.namespace.name"])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if ServiceGraphNodeAttributesForServer(nil) != nil {
			t.Error("nil attrs should return nil")
		}
		if ServiceGraphNodeAttributesForServer(map[string]string{}) != nil {
			t.Error("empty map should return nil")
		}
	})

	t.Run("only bare server label", func(t *testing.T) {
		if got := ServiceGraphNodeAttributesForServer(map[string]string{"server": "x"}); got != nil {
			t.Errorf("want nil when only redundant server label, got %v", got)
		}
	})
}

func TestEdgeToModelNodeAttributes(t *testing.T) {
	edge := collectormetrics.ServiceGraphEdge{
		ToNodeIsVirtual: true,
		RequestCount:    5,
		LastUpdated:     time.Unix(1000, 0).UTC(),
		Attributes: map[string]string{
			"client":           "a",
			"server":           "b",
			"server_db_system": "redis",
		},
	}
	m := EdgeToModel("b|extra", edge, ServiceGraphNodeAttributesForServer(edge.Attributes))
	if len(m.NodeAttributes) != 1 {
		t.Fatalf("want 1 nodeAttribute (db.system only; bare server omitted), got %+v", m.NodeAttributes)
	}
	keys := make(map[string]string)
	for _, a := range m.NodeAttributes {
		keys[a.Key] = a.Value
	}
	if keys["server"] != "" || keys["server_db_system"] != "" || keys["db.system"] != "redis" {
		t.Errorf("unexpected projection: %v", keys)
	}
	if keys["client"] != "" {
		t.Error("client should not appear in server-side projection")
	}
}
