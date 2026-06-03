package config

import "testing"

func baseProxyConfig() *Config {
	return &Config{
		Exporters: GenericMap{
			"otlp_http/dynatrace-x": GenericMap{"endpoint": "https://x.live.dynatrace.com/api/v2/otlp"},
			"otlp/grpcbackend":      GenericMap{"endpoint": "backend:4317"},
			"kafka/logs":            GenericMap{"brokers": []string{"b:9092"}},
		},
		Service: Service{Pipelines: map[string]Pipeline{
			"traces/dynatrace-x": {Exporters: []string{"otlp_http/dynatrace-x"}},
			"traces/grpcbackend": {Exporters: []string{"otlp/grpcbackend"}},
			"logs/kafka":         {Exporters: []string{"kafka/logs"}},
		}},
	}
}

func TestInjectProxy_HTTPGetsProxyURLAndCA(t *testing.T) {
	cfg := baseProxyConfig()
	if err := InjectProxy(cfg, "http://proxy.corp.local:8080", "/etc/odigos/ca.pem"); err != nil {
		t.Fatalf("InjectProxy: %v", err)
	}
	em := asGenericMap(cfg.Exporters["otlp_http/dynatrace-x"])
	if em["proxy_url"] != "http://proxy.corp.local:8080" {
		t.Fatalf("http exporter missing proxy_url: %+v", em)
	}
	tls := asGenericMap(em["tls"])
	if tls == nil || tls["ca_file"] != "/etc/odigos/ca.pem" {
		t.Fatalf("http exporter missing tls.ca_file: %+v", em["tls"])
	}
}

func TestInjectProxy_GRPCRetypedAndPipelineRewired(t *testing.T) {
	cfg := baseProxyConfig()
	if err := InjectProxy(cfg, "http://proxy:8080", ""); err != nil {
		t.Fatalf("InjectProxy: %v", err)
	}
	if _, stillThere := cfg.Exporters["otlp/grpcbackend"]; stillThere {
		t.Fatal("otlp/ exporter should have been re-typed away")
	}
	em := asGenericMap(cfg.Exporters["otlpproxygrpc/grpcbackend"])
	if em == nil || em["proxy_url"] != "http://proxy:8080" {
		t.Fatalf("grpc exporter not re-typed with proxy_url: %+v", cfg.Exporters)
	}
	if got := cfg.Service.Pipelines["traces/grpcbackend"].Exporters; len(got) != 1 || got[0] != "otlpproxygrpc/grpcbackend" {
		t.Fatalf("pipeline not rewired to otlpproxygrpc: %v", got)
	}
}

func TestInjectProxy_LeavesNonOTLPUntouched(t *testing.T) {
	cfg := baseProxyConfig()
	_ = InjectProxy(cfg, "http://proxy:8080", "")
	em := asGenericMap(cfg.Exporters["kafka/logs"])
	if _, has := em["proxy_url"]; has {
		t.Fatal("non-OTLP exporter must not get proxy_url")
	}
}

func TestInjectProxy_RejectsBadURL(t *testing.T) {
	for _, bad := range []string{`"http://proxy:8080"`, "ftp://proxy:21", "http://proxy:8080 ", "://nohost"} {
		if err := InjectProxy(baseProxyConfig(), bad, ""); err == nil {
			t.Fatalf("expected rejection for %q", bad)
		}
	}
}
