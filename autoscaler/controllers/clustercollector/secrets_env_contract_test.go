package clustercollector

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	cfg "github.com/odigos-io/odigos/common/config"
)

// The gateway deployment mounts a destination Secret with an envFrom prefix while the collector
// config placeholders that read those env vars are rendered by common/config. Nothing ties the
// two together at compile time, so if either side changes how it derives the env var name, the
// exporter silently expands to an unset variable. These tests drive both sides from one
// Destination object and assert they agree.

var secretPlaceholderPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

func destinationWithSecret(id string, destType common.DestinationType, config map[string]string, signals []common.ObservabilitySignal) odigosv1.Destination {
	return odigosv1.Destination{
		ObjectMeta: metav1.ObjectMeta{Name: id},
		Spec: odigosv1.DestinationSpec{
			Type:            destType,
			Data:            config,
			Signals:         signals,
			DestinationName: id,
			SecretRef:       &corev1.LocalObjectReference{Name: id + "-secret"},
		},
	}
}

func renderDestinationConfig(t *testing.T, dest odigosv1.Destination) *cfg.Config {
	t.Helper()

	configers, err := cfg.LoadConfigers()
	require.NoError(t, err)
	configer, exists := configers[dest.Spec.Type]
	require.True(t, exists)

	collectorConfig := &cfg.Config{
		Exporters:  cfg.GenericMap{},
		Processors: cfg.GenericMap{},
		Extensions: cfg.GenericMap{},
		Service:    cfg.Service{Pipelines: map[string]cfg.Pipeline{}},
	}
	_, err = configer.ModifyConfig(dest, collectorConfig)
	require.NoError(t, err)

	return collectorConfig
}

func collectSecretPlaceholders(value any, found *[]string) {
	switch v := value.(type) {
	case string:
		for _, match := range secretPlaceholderPattern.FindAllStringSubmatch(v, -1) {
			*found = append(*found, match[1])
		}
	case cfg.GenericMap:
		for _, nested := range v {
			collectSecretPlaceholders(nested, found)
		}
	case map[string]any:
		for _, nested := range v {
			collectSecretPlaceholders(nested, found)
		}
	case []any:
		for _, nested := range v {
			collectSecretPlaceholders(nested, found)
		}
	}
}

func renderedPlaceholders(collectorConfig *cfg.Config) []string {
	var found []string
	collectSecretPlaceholders(collectorConfig.Exporters, &found)
	collectSecretPlaceholders(collectorConfig.Extensions, &found)
	collectSecretPlaceholders(collectorConfig.Processors, &found)

	seen := map[string]struct{}{}
	unique := make([]string, 0, len(found))
	for _, placeholder := range found {
		if _, ok := seen[placeholder]; ok {
			continue
		}
		seen[placeholder] = struct{}{}
		unique = append(unique, placeholder)
	}
	sort.Strings(unique)
	return unique
}

func TestMountedSecretPrefixResolvesTheRenderedConfigPlaceholders(t *testing.T) {
	tests := []struct {
		name string
		dest odigosv1.Destination
		// the keys the destination Secret must hold for the rendered placeholders to resolve.
		wantSecretKeys []string
	}{
		{
			name: "single secret field",
			dest: destinationWithSecret("odigos.io.dest.datadog-aaaa", common.DatadogDestinationType,
				map[string]string{"DATADOG_SITE": "datadoghq.com"},
				[]common.ObservabilitySignal{common.TracesObservabilitySignal}),
			wantSecretKeys: []string{"DATADOG_API_KEY"},
		},
		{
			name: "one secret field per signal",
			dest: destinationWithSecret("odigos.io.dest.logzio-aaaa", common.LogzioDestinationType,
				map[string]string{"LOGZIO_REGION": "us"},
				[]common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}),
			wantSecretKeys: []string{"LOGZIO_LOGS_TOKEN", "LOGZIO_METRICS_TOKEN", "LOGZIO_TRACING_TOKEN"},
		},
		{
			name: "mtls certificate and key",
			dest: destinationWithSecret("odigos.io.dest.otlphttp-aaaa", common.OtlpHttpDestinationType,
				map[string]string{
					"OTLP_HTTP_ENDPOINT":     "https://otlphttp.example.com",
					"OTLP_HTTP_TLS_ENABLED":  "true",
					"OTLP_HTTP_MTLS_ENABLED": "true",
				},
				[]common.ObservabilitySignal{common.TracesObservabilitySignal}),
			wantSecretKeys: []string{"OTLP_HTTP_CLIENT_CERT_PEM", "OTLP_HTTP_CLIENT_KEY_PEM"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := getSecretsFromDests(&odigosv1.DestinationList{Items: []odigosv1.Destination{tt.dest}})
			require.Len(t, sources, 1)
			prefix := sources[0].Prefix
			require.NotEmpty(t, prefix)
			assert.Equal(t, tt.dest.Spec.SecretRef.Name, sources[0].SecretRef.Name)

			placeholders := renderedPlaceholders(renderDestinationConfig(t, tt.dest))
			require.NotEmpty(t, placeholders)

			var secretKeys []string
			for _, placeholder := range placeholders {
				require.True(t, strings.HasPrefix(placeholder, prefix),
					"%q is not resolved by the env vars the gateway mounts with prefix %q", placeholder, prefix)
				secretKeys = append(secretKeys, strings.TrimPrefix(placeholder, prefix))
			}
			// the remainder is the Secret key as the user stored it, so prefix + key is exactly
			// the env var name the gateway container ends up with.
			assert.Equal(t, tt.wantSecretKeys, secretKeys)
		})
	}
}

func TestDynamicDestinationPlaceholdersResolveFromAnUnprefixedSecret(t *testing.T) {
	dest := destinationWithSecret("odigos.io.dest.dynamic-aaaa", common.DynamicDestinationType,
		map[string]string{
			"DYNAMIC_DESTINATION_TYPE":   "otlphttp",
			"DYNAMIC_CONFIGURATION_DATA": "endpoint: https://dynamic.example.com\ntoken: ${MY_OWN_TOKEN}\n",
		},
		[]common.ObservabilitySignal{common.TracesObservabilitySignal})

	sources := getSecretsFromDests(&odigosv1.DestinationList{Items: []odigosv1.Destination{dest}})
	require.Len(t, sources, 1)
	assert.Empty(t, sources[0].Prefix)

	// the raw exporter yaml is written by the user, so its placeholder must survive verbatim -
	// prefixing the mounted Secret would leave MY_OWN_TOKEN unset.
	assert.Equal(t, []string{"MY_OWN_TOKEN"}, renderedPlaceholders(renderDestinationConfig(t, dest)))
}

func TestGetSecretsFromDestsSkipsDestinationsWithoutASecret(t *testing.T) {
	withSecret := destinationWithSecret("odigos.io.dest.datadog-aaaa", common.DatadogDestinationType, nil, nil)
	withoutSecret := odigosv1.Destination{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.jaeger-bbbb"},
		Spec:       odigosv1.DestinationSpec{Type: common.JaegerDestinationType},
	}

	assert.Empty(t, getSecretsFromDests(&odigosv1.DestinationList{}))
	assert.Empty(t, getSecretsFromDests(&odigosv1.DestinationList{Items: []odigosv1.Destination{withoutSecret}}))

	sources := getSecretsFromDests(&odigosv1.DestinationList{Items: []odigosv1.Destination{withoutSecret, withSecret}})
	require.Len(t, sources, 1)
	assert.Equal(t, "odigos.io.dest.datadog-aaaa-secret", sources[0].SecretRef.Name)
}
