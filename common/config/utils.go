package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
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

func parseOtlpHttpEndpoint(rawUrl string, defaultPort string, requiredPath string) (string, error) {
	noWhiteSpaces := strings.TrimSpace(rawUrl)
	parsedUrl, err := url.Parse(noWhiteSpaces)
	if err != nil {
		return "", fmt.Errorf("failed to parse otlp http endpoint: %w", err)
	}
	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
		return "", fmt.Errorf("invalid otlp http endpoint scheme: %s", parsedUrl.Scheme)
	}

	// A defailt port exists, and the parsed URL does not have a port
	if defaultPort != "" && parsedUrl.Port() == "" {
		// Append the default port
		parsedUrl.Host = net.JoinHostPort(parsedUrl.Hostname(), defaultPort)
	}

	if parsedUrl.Path == "" {
		// Path is empty, append the required path
		parsedUrl.Path = requiredPath
	} else if requiredPath != "" && parsedUrl.Path != requiredPath {
		// Path already exists, and is not equal to the required path
		return "", fmt.Errorf("invalid otlp http endpoint path: %s", parsedUrl.Path)
	}

	return parsedUrl.String(), nil
}

func urlHostContainsPort(host string) bool {
	lastIndex := strings.LastIndex(host, "]")
	if lastIndex != -1 { // ipv6
		return lastIndex+1 < len(host) && host[lastIndex+1] == ':'
	} else { // dns host or ipv4
		return strings.Contains(host, ":")
	}
}

func getBooleanConfig(currentValue string, deprecatedValue string) bool {
	lowerCaseValue := strings.ToLower(currentValue)
	return lowerCaseValue == "true" || lowerCaseValue == deprecatedValue
}

func parseBool(value string) bool {
	result, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return result
}

func parseInt(value string) int {
	num, err := strconv.Atoi(value)
	if err != nil {
		panic(err.Error())
	}
	return num
}

func errorMissingKey(key string) error {
	return fmt.Errorf("key (\"%q\") not specified, destination will not be configured", key)
}

type SpanMetricNames struct {
	SpanMetricsConnector string
	TracesPipeline       string
}

// This function configures a connector that converts trace-spans to metrics.
// This is meant for destination that accept metrics but not traces!
func applySpanMetricsConnector(currentConfig *Config, uniqueUri string) SpanMetricNames {
	spanMetricsConnectorName := "spanmetrics/" + uniqueUri
	tracesPipelineName := "traces/spanmetrics-" + uniqueUri

	// Send SpanMetrics to prometheus
	// configure a connector which will convert spans to metrics, this should ideally be configurable,
	// and available for all metrics destinations
	// TODO: this should be an action ("SpanMetrics connector")?
	currentConfig.Connectors[spanMetricsConnectorName] = GenericMap{
		"histogram": GenericMap{
			"explicit": GenericMap{
				"buckets": []string{"100us", "1ms", "2ms", "6ms", "10ms", "100ms", "250ms"},
			},
		},
		// Taking into account changes in the semantic conventions, to support a range of instrumentation libraries
		"dimensions": []GenericMap{
			{
				"name": "http.method",
			},
			{
				"name": "http.request.method",
			},
			{
				"name": "http.status_code",
			},
			{
				"name": "http.response.status_code",
			},
			{
				"name": "http.route",
			},
		},
		"exemplars": GenericMap{
			"enabled": true,
		},
		"exclude_dimensions":              []string{"status.code"},
		"dimensions_cache_size":           1000,
		"aggregation_temporality":         "AGGREGATION_TEMPORALITY_CUMULATIVE",
		"metrics_flush_interval":          "15s",
		"metrics_expiration":              "5m",
		"resource_metrics_key_attributes": []string{"service.name", "telemetry.sdk.language", "telemetry.sdk.name"},
		"events": GenericMap{
			"enabled": true,
			"dimensions": []GenericMap{
				{
					"name": "exception.type",
				},
				{
					"name": "exception.message",
				},
			},
		},
	}

	currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
		Exporters: []string{spanMetricsConnectorName},
	}

	return SpanMetricNames{
		SpanMetricsConnector: spanMetricsConnectorName,
		TracesPipeline:       tracesPipelineName,
	}
}
