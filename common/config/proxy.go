package config

import (
	"fmt"
	"net/url"
	"strings"
)

// InjectProxy routes every OTLP exporter in cfg through an egress proxy. It is
// the single, destination-agnostic hook both control planes (the K8s autoscaler
// and the vm-agent) call after assembling exporters, so proxy configuration is
// never wired per destination.
//
//   - otlphttp* / otlp_http* exporters: confighttp supports proxy_url natively,
//     so we set proxy_url (+ tls.ca_file when a CA is supplied).
//   - otlp* (gRPC) exporters: configgrpc has no proxy field, so the exporter is
//     re-typed to otlpproxygrpc (the custom CONNECT-dialer exporter) and every
//     pipeline reference is rewired to the new name.
//
// Non-OTLP exporters (Kafka, S3, vendor SDKs) are left untouched. When the proxy
// is disabled the caller simply does not invoke InjectProxy, so exporters stay
// direct — the toggle is "call it or don't", and the URL stays stored upstream.
func InjectProxy(cfg *Config, proxyURL, caFile string) error {
	if cfg == nil || len(cfg.Exporters) == 0 {
		return nil
	}
	if err := validateProxyURL(proxyURL); err != nil {
		return err
	}

	renames := map[string]string{}
	for name, raw := range cfg.Exporters {
		em := asGenericMap(raw)
		if em == nil {
			continue
		}
		switch {
		case isHTTPOTLPExporter(name):
			em["proxy_url"] = proxyURL
			applyProxyCA(em, caFile)
			cfg.Exporters[name] = em
		case isGRPCOTLPExporter(name):
			em["proxy_url"] = proxyURL
			applyProxyCA(em, caFile)
			newName := "otlpproxygrpc" + exporterNameSuffix(name)
			delete(cfg.Exporters, name)
			cfg.Exporters[newName] = em
			renames[name] = newName
		}
	}
	if len(renames) > 0 {
		rewirePipelineExporters(cfg, renames)
	}
	return nil
}

// validateProxyURL rejects malformed proxy URLs (the stray-quote outage class)
// and unsupported schemes. An empty URL is invalid here — callers must only
// invoke InjectProxy when a proxy is actually configured.
func validateProxyURL(proxyURL string) error {
	if strings.TrimSpace(proxyURL) != proxyURL || strings.ContainsAny(proxyURL, "\"'") {
		return fmt.Errorf("proxy url must not contain quotes or surrounding whitespace, got %q", proxyURL)
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("invalid proxy url %q: %w", proxyURL, err)
	}
	switch u.Scheme {
	case "http", "https", "socks5":
	default:
		return fmt.Errorf("proxy url scheme must be http, https or socks5, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("proxy url %q is missing host:port", proxyURL)
	}
	return nil
}

func isHTTPOTLPExporter(name string) bool {
	return strings.HasPrefix(name, "otlphttp") || strings.HasPrefix(name, "otlp_http")
}

func isGRPCOTLPExporter(name string) bool {
	return name == "otlp" || strings.HasPrefix(name, "otlp/")
}

// exporterNameSuffix returns the "/instance" part of an exporter key (incl. the
// slash), or "" for a bare type. e.g. "otlp/dynatrace-x" -> "/dynatrace-x".
func exporterNameSuffix(name string) string {
	if i := strings.IndexByte(name, '/'); i >= 0 {
		return name[i:]
	}
	return ""
}

func applyProxyCA(em GenericMap, caFile string) {
	if caFile == "" {
		return
	}
	tls := asGenericMap(em["tls"])
	if tls == nil {
		tls = GenericMap{}
	}
	tls["ca_file"] = caFile
	em["tls"] = tls
}

func rewirePipelineExporters(cfg *Config, renames map[string]string) {
	for pName, p := range cfg.Service.Pipelines {
		changed := false
		for i, e := range p.Exporters {
			if nn, ok := renames[e]; ok {
				p.Exporters[i] = nn
				changed = true
			}
		}
		if changed {
			cfg.Service.Pipelines[pName] = p
		}
	}
}

// asGenericMap coerces the GenericMap or map[string]interface{} forms a value
// may take (in-process vs after a YAML/JSON round-trip).
func asGenericMap(v interface{}) GenericMap {
	switch m := v.(type) {
	case GenericMap:
		return m
	case map[string]interface{}:
		return GenericMap(m)
	default:
		return nil
	}
}
