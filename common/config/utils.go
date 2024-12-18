package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func parseUnencryptedOtlpGrpcUrl(rawURL string) (grpcUrl string, err error) {

	rawURL = strings.TrimSpace(rawURL)
	urlWithScheme := rawURL

	// if no scheme is provided, we default to grpc.
	// this is only for the purpose of parsing, we will ignore it later on
	if !strings.Contains(rawURL, "://") {
		urlWithScheme = "grpc://" + rawURL
	}

	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	if parsedUrl.Scheme == "https" || parsedUrl.Scheme == "grpcs" {
		return "", fmt.Errorf("grpc endpoint does not support tls")
	}

	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "grpc" {
		return "", fmt.Errorf("unexpected scheme %s", parsedUrl.Scheme)
	}

	// validate no path is provided, as this indicates using improper url (like otlp http with /v1/traces path)
	if parsedUrl.Path != "" {
		return "", fmt.Errorf("unexpected path for grpc endpoint %s", parsedUrl.Path)
	}

	// validate no query is provided, as this indicates using improper endpoint
	if parsedUrl.RawQuery != "" {
		return "", fmt.Errorf("unexpected query for grpc endpoint %s", parsedUrl.RawQuery)
	}

	// in grpc endpoint, there is no user or password
	if parsedUrl.User != nil {
		return "", fmt.Errorf("unexpected user info for grpc endpoint %s", parsedUrl.User)
	}

	// we default to port 4317 for otlp grpc.
	// if missing we add it.
	hostport := parsedUrl.Host
	if !urlHostContainsPort(hostport) {
		hostport += ":4317"
	}

	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", err
	}

	if host == "" {
		return "", fmt.Errorf("missing host in grpc endpoint %s", rawURL)
	}

	// Check if the host is an IPv6 address and enclose it in square brackets
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}

	return fmt.Sprintf("%s:%s", host, port), nil
}

// parseEncryptedOtlpGrpcUrl parses and validates a URL for an encrypted OTLP gRPC endpoint.
// It ensures the scheme is either "https" or "grpcs", and appends the default OTLP gRPC port (4317) if missing.
func parseEncryptedOtlpGrpcUrl(rawURL string) (grpcUrl string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	urlWithScheme := rawURL

	// If no scheme is provided, we assume "grpcs" for encrypted traffic.
	if !strings.Contains(rawURL, "://") {
		urlWithScheme = "grpcs://" + rawURL
	}

	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	// Validate scheme to ensure it is for encrypted gRPC.
	if parsedUrl.Scheme != "https" && parsedUrl.Scheme != "grpcs" {
		return "", fmt.Errorf("unexpected scheme %s for encrypted gRPC endpoint", parsedUrl.Scheme)
	}

	// Ensure no path is provided.
	if parsedUrl.Path != "" {
		return "", fmt.Errorf("unexpected path for encrypted gRPC endpoint %s", parsedUrl.Path)
	}

	// Ensure no query parameters are provided.
	if parsedUrl.RawQuery != "" {
		return "", fmt.Errorf("unexpected query for encrypted gRPC endpoint %s", parsedUrl.RawQuery)
	}

	// Ensure no user info is provided.
	if parsedUrl.User != nil {
		return "", fmt.Errorf("unexpected user info for encrypted gRPC endpoint %s", parsedUrl.User)
	}

	// Add the default OTLP gRPC port (4317) if missing.
	hostport := parsedUrl.Host
	if !urlHostContainsPort(hostport) {
		hostport += ":4317"
	}

	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", err
	}

	if host == "" {
		return "", fmt.Errorf("missing host in encrypted gRPC endpoint %s", rawURL)
	}

	// Check if the host is an IPv6 address and enclose it in square brackets.
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}

	// Construct the final gRPC URL.
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
