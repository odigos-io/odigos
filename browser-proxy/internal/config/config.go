// Package config loads the odigos-browser-proxy sidecar configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Environment variable names. These MUST match the constants in
// github.com/odigos-io/odigos/api/k8sconsts (browserproxy.go), which the instrumentor webhook
// uses when injecting the sidecar. They are duplicated here so the sidecar stays a tiny,
// dependency-free module rather than pulling in the api/k8s module graph.
const (
	envUpstream           = "ODIGOS_BROWSER_PROXY_UPSTREAM"
	envListenAddr         = "ODIGOS_BROWSER_PROXY_LISTEN_ADDR"
	envAgentDir           = "ODIGOS_BROWSER_PROXY_AGENT_DIR"
	envAgentFile          = "ODIGOS_BROWSER_PROXY_AGENT_FILE"
	envOtlpHTTPEndpoint   = "ODIGOS_BROWSER_PROXY_OTLP_HTTP_ENDPOINT"
	envServiceName        = "ODIGOS_BROWSER_PROXY_SERVICE_NAME"
	envResourceAttributes = "ODIGOS_BROWSER_PROXY_RESOURCE_ATTRIBUTES"
	envPropagateCorsUrls  = "ODIGOS_BROWSER_PROXY_PROPAGATE_CORS_URLS"
	envAppPort            = "ODIGOS_BROWSER_PROXY_APP_PORT"
	envProxyUID           = "ODIGOS_BROWSER_PROXY_UID"
)

// Defaults, also mirrored from api/k8sconsts.
const (
	DefaultListenPort = 15001
	DefaultAgentDir   = "/var/odigos/browser"
	DefaultAgentFile  = "agent.js"

	// Same-origin paths the sidecar reserves for itself.
	PathPrefix     = "/__odigos/"
	AgentJsPath    = "/__odigos/agent.js"
	HealthPath     = "/__odigos/healthz"
	TracesPath     = "/__odigos/v1/traces"
	OtlpPathPrefix = "/__odigos/v1/"
)

// Config holds the resolved sidecar configuration.
type Config struct {
	// ListenAddr is the address the sidecar's HTTP server binds to (e.g. ":15001").
	ListenAddr string
	// Upstream is the application base URL the sidecar forwards browser requests to.
	Upstream string
	// AgentDir is the directory the browser SDK bundle is served from.
	AgentDir string
	// AgentFile is the file name of the browser SDK bundle within AgentDir.
	AgentFile string
	// OtlpHTTPEndpoint is the base OTLP/HTTP endpoint of the node-local collector.
	OtlpHTTPEndpoint string
	// ServiceName is reported as service.name by the browser SDK.
	ServiceName string
	// ResourceAttributes is an OTEL_RESOURCE_ATTRIBUTES-style string (k=v,k2=v2).
	ResourceAttributes string
	// PropagateCorsUrls is a comma-separated list of URLs/regexes for trace-context propagation.
	PropagateCorsUrls string
}

// LoadServe loads and validates the configuration needed to run the proxy server.
func LoadServe() (*Config, error) {
	cfg := &Config{
		ListenAddr:         getenvDefault(envListenAddr, fmt.Sprintf(":%d", DefaultListenPort)),
		Upstream:           os.Getenv(envUpstream),
		AgentDir:           getenvDefault(envAgentDir, DefaultAgentDir),
		AgentFile:          getenvDefault(envAgentFile, DefaultAgentFile),
		OtlpHTTPEndpoint:   strings.TrimRight(os.Getenv(envOtlpHTTPEndpoint), "/"),
		ServiceName:        os.Getenv(envServiceName),
		ResourceAttributes: os.Getenv(envResourceAttributes),
		PropagateCorsUrls:  os.Getenv(envPropagateCorsUrls),
	}

	if cfg.Upstream == "" {
		return nil, fmt.Errorf("%s is required", envUpstream)
	}
	if cfg.OtlpHTTPEndpoint == "" {
		return nil, fmt.Errorf("%s is required", envOtlpHTTPEndpoint)
	}

	return cfg, nil
}

// InitConfig holds the configuration for the iptables init mode.
type InitConfig struct {
	AppPort   int
	ProxyPort int
	ProxyUID  int
}

// LoadInit loads and validates the configuration needed to apply the iptables redirect.
func LoadInit() (*InitConfig, error) {
	appPort, err := strconv.Atoi(os.Getenv(envAppPort))
	if err != nil || appPort <= 0 || appPort > 65535 {
		return nil, fmt.Errorf("%s must be a valid TCP port, got %q", envAppPort, os.Getenv(envAppPort))
	}

	proxyUID, err := strconv.Atoi(os.Getenv(envProxyUID))
	if err != nil || proxyUID < 0 {
		return nil, fmt.Errorf("%s must be a valid UID, got %q", envProxyUID, os.Getenv(envProxyUID))
	}

	proxyPort := DefaultListenPort
	if v := os.Getenv(envListenAddr); v != "" {
		// ListenAddr is ":15001"; extract the port.
		if idx := strings.LastIndex(v, ":"); idx >= 0 {
			if p, perr := strconv.Atoi(v[idx+1:]); perr == nil && p > 0 {
				proxyPort = p
			}
		}
	}

	return &InitConfig{AppPort: appPort, ProxyPort: proxyPort, ProxyUID: proxyUID}, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
