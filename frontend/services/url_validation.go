package services

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// ValidateURLForTestConnection checks if a URL is allowed for test connection based on the AllowedDomains configuration.
func ValidateURLForTestConnection(ctx context.Context, testURL string) error {
	cfg, err := getOdigosConfiguration(ctx)
	if err != nil {
		return nil
	}

	if len(cfg.AllowedTestConnectionHosts) == 0 {
		return nil
	}

	if slices.Contains(cfg.AllowedTestConnectionHosts, "*") {
		return nil
	}

	parsedURL, err := url.Parse(testURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	urlDomain := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	for _, allowedDomain := range cfg.AllowedTestConnectionHosts {
		normalizedAllowedDomain := normalizeDomain(allowedDomain)

		if urlDomain == normalizedAllowedDomain {
			return nil
		}

		if domainWithoutWildcard, ok := strings.CutPrefix(normalizedAllowedDomain, "*."); ok {
			if strings.HasSuffix(parsedURL.Host, "."+domainWithoutWildcard) {
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

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)

	if !strings.Contains(domain, "://") {
		domain = "https://" + domain
	}

	return domain
}
