package services

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var testConnectionEndpointFields = map[string]struct{}{
	"OTLP_HTTP_ENDPOINT":          {},
	"OTLP_HTTP_TRACES_ENDPOINT":   {},
	"OTLP_HTTP_METRICS_ENDPOINT":  {},
	"OTLP_HTTP_LOGS_ENDPOINT":     {},
	"OTLP_HTTP_PROFILES_ENDPOINT": {},
	"OTLP_GRPC_ENDPOINT":          {},
}

type testConnectionDomain struct {
	scheme string
	host   string
}

func validateURLAgainstAllowedHosts(testURL string, allowedHosts []string) error {
	if len(allowedHosts) == 0 {
		return nil
	}

	if slices.Contains(allowedHosts, "*") {
		return nil
	}

	urlDomain, err := parseTestConnectionDomain(testURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	for _, allowedDomain := range allowedHosts {
		normalizedAllowedDomain, err := parseTestConnectionDomain(allowedDomain)
		if err != nil {
			continue
		}

		if urlDomain == normalizedAllowedDomain {
			return nil
		}

		if domainWithoutWildcard, ok := strings.CutPrefix(normalizedAllowedDomain.host, "*."); ok {
			if urlDomain.scheme == normalizedAllowedDomain.scheme && strings.HasSuffix(urlDomain.host, "."+domainWithoutWildcard) {
				return nil
			}
		}
	}

	return fmt.Errorf("URL '%s' is not in the allowed domains list. Please contact your administrator to add this domain to the allowed list.", testURL)
}

func getOdigosConfiguration(ctx context.Context) (*common.OdigosConfiguration, error) {
	ns := env.GetCurrentNamespace()

	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get odigos configuration: %w", err)
	}

	var cfg common.OdigosConfiguration
	err = yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal odigos configuration: %w", err)
	}

	return &cfg, nil
}

func parseTestConnectionDomain(domain string) (testConnectionDomain, error) {
	normalizedDomain := strings.TrimSpace(domain)

	if !strings.Contains(normalizedDomain, "://") {
		normalizedDomain = "https://" + normalizedDomain
	}

	parsedURL, err := url.Parse(normalizedDomain)
	if err != nil {
		return testConnectionDomain{}, err
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return testConnectionDomain{}, fmt.Errorf("missing scheme or host")
	}

	return testConnectionDomain{
		scheme: parsedURL.Scheme,
		host:   parsedURL.Host,
	}, nil
}

func ValidateDestinationURLs(ctx context.Context, destination model.DestinationInput) error {
	cfg, err := getOdigosConfiguration(ctx)
	if err != nil {
		return nil
	}

	return validateDestinationURLsAgainstAllowedHosts(destination, cfg.AllowedTestConnectionHosts)
}

func validateDestinationURLsAgainstAllowedHosts(destination model.DestinationInput, allowedHosts []string) error {
	for _, field := range destination.Fields {
		if _, ok := testConnectionEndpointFields[field.Key]; ok && field.Value != "" {
			err := validateURLAgainstAllowedHosts(field.Value, allowedHosts)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
