package otlpproxygrpcexporter

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config is the OTLP/gRPC exporter config plus a proxy_url. It mirrors the
// upstream otlp exporter (embedded configgrpc.ClientConfig + standard
// timeout/queue/retry) and adds proxy_url, which configgrpc itself does not
// support for gRPC. When proxy_url is set, the gRPC connection is established
// through an HTTP CONNECT tunnel to that proxy; when empty, this exporter
// behaves exactly like the stock otlp exporter (direct connection).
type Config struct {
	TimeoutConfig exporterhelper.TimeoutConfig                            `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig   configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig                               `mapstructure:"retry_on_failure"`

	configgrpc.ClientConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// ProxyURL routes the gRPC connection through an HTTP CONNECT proxy
	// (e.g. http://proxy.corp.local:8080). Supports http/https/socks5 schemes
	// and optional user:pass@ credentials. Empty disables proxying.
	ProxyURL string `mapstructure:"proxy_url"`
}

var _ component.Config = (*Config)(nil)

// Validate checks the endpoint and, when set, that proxy_url is a well-formed
// proxy URL with a supported scheme. Rejecting a malformed value here is what
// prevents the classic "proxy stored with stray quotes" outage.
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint must be specified")
	}
	if c.ProxyURL == "" {
		return nil
	}
	raw := strings.TrimSpace(c.ProxyURL)
	if raw != c.ProxyURL || strings.ContainsAny(raw, "\"'") {
		return fmt.Errorf("proxy_url must not contain quotes or surrounding whitespace, got %q", c.ProxyURL)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid proxy_url %q: %w", c.ProxyURL, err)
	}
	switch u.Scheme {
	case "http", "https", "socks5":
	default:
		return fmt.Errorf("proxy_url scheme must be http, https or socks5, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("proxy_url %q is missing host:port", c.ProxyURL)
	}
	return nil
}
