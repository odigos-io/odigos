package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func parseOtlpGrpcUrl(rawURL string, encrypted bool) (grpcUrl string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	urlWithScheme := rawURL

	// Default scheme based on encryption flag
	defaultScheme := "grpc"
	if encrypted {
		defaultScheme = "grpcs"
	}

	// Add scheme if not provided
	if !strings.Contains(rawURL, "://") {
		urlWithScheme = defaultScheme + "://" + rawURL
	}

	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	// Validate scheme based on encryption flag
	validSchemes := map[string]struct{}{
		"grpc":  {},
		"http":  {},
		"grpcs": {},
		"https": {},
	}

	if encrypted {
		if _, ok := validSchemes[parsedUrl.Scheme]; !ok || (parsedUrl.Scheme != "https" && parsedUrl.Scheme != "grpcs") {
			return "", fmt.Errorf("unexpected scheme %s for encrypted gRPC endpoint", parsedUrl.Scheme)
		}
	} else {
		if parsedUrl.Scheme == "https" || parsedUrl.Scheme == "grpcs" {
			return "", fmt.Errorf("grpc endpoint does not support TLS")
		}
		if _, ok := validSchemes[parsedUrl.Scheme]; !ok {
			return "", fmt.Errorf("unexpected scheme %s for unencrypted gRPC endpoint", parsedUrl.Scheme)
		}
	}

	// Validate URL components
	if parsedUrl.Path != "" {
		return "", fmt.Errorf("unexpected path for gRPC endpoint %s", parsedUrl.Path)
	}

	if parsedUrl.RawQuery != "" {
		return "", fmt.Errorf("unexpected query for gRPC endpoint %s", parsedUrl.RawQuery)
	}

	if parsedUrl.User != nil {
		return "", fmt.Errorf("unexpected user info for gRPC endpoint %s", parsedUrl.User)
	}

	// Add default port if missing
	hostport := parsedUrl.Host
	if !urlHostContainsPort(hostport) {
		hostport += ":4317"
	}

	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", err
	}

	if host == "" {
		return "", fmt.Errorf("missing host in gRPC endpoint %s", rawURL)
	}

	// Enclose IPv6 addresses in square brackets
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}

	return fmt.Sprintf("%s:%s", host, port), nil
}

func urlHostContainsPort(host string) bool {
	lastIndex := strings.LastIndex(host, "]")
	if lastIndex != -1 { // ipv6
		return lastIndex+1 < len(host) && host[lastIndex+1] == ':'
	} else { // dns host or ipv4
		return strings.Contains(host, ":")
	}
}
