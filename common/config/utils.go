package config

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

func encodeBase64(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func decodeBase64(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getSecret(secretName string) (*corev1.Secret, error) {
	ns := env.GetCurrentNamespace()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return &v1.Secret{}, fmt.Errorf("failed to load in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return &v1.Secret{}, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	secret, err := clientset.CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return &v1.Secret{}, fmt.Errorf("failed to get secret %s/%s: %w", ns, secretName, err)
	}

	return secret, nil
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
