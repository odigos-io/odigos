package odigosconfigk8sextension

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/odigos-io/odigos/common/consts"
)

// instrumentationConfig builds the InstrumentationConfig the instrumentor writes for a source
// with a PII masking action applied to one container.
func instrumentationConfig(icName, namespace, containerName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "odigos.io/v1alpha1",
		"kind":       "InstrumentationConfig",
		"metadata": map[string]any{
			"name":      icName,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"workloadCollectorConfig": []any{
				map[string]any{
					"containerName": containerName,
					"piiMasking": map[string]any{
						"piiCategories": []any{"CREDIT_CARD"},
					},
				},
			},
		},
	}}
}

// The config the instrumentor writes for a source must be reachable from the resource attributes
// of that source's spans, otherwise every processor reading per-source config skips the workload.
func TestGetFromResource_ResolvesConfigForEverySourceKind(t *testing.T) {
	for _, tc := range []struct {
		name   string
		icName string
		attrs  map[string]string
	}{
		{
			name:   "deployment",
			icName: "deployment-checkout",
			attrs: map[string]string{
				string(semconv.K8SDeploymentNameKey): "checkout",
				consts.OdigosWorkloadKindAttribute:   "Deployment",
				consts.OdigosWorkloadNameAttribute:   "checkout",
			},
		},
		{
			name:   "cronjob",
			icName: "cronjob-backup",
			attrs: map[string]string{
				string(semconv.K8SCronJobNameKey):  "backup",
				string(semconv.K8SJobNameKey):      "backup-28812345",
				consts.OdigosWorkloadKindAttribute: "CronJob",
				consts.OdigosWorkloadNameAttribute: "backup",
			},
		},
		{
			name:   "deploymentconfig",
			icName: "deploymentconfig-frontend",
			attrs: map[string]string{
				string(semconv.K8SDeploymentNameKey): "frontend",
				consts.OdigosWorkloadKindAttribute:   "DeploymentConfig",
				consts.OdigosWorkloadNameAttribute:   "frontend",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ext := &OdigosWorkloadConfig{cache: newCache(), logger: zap.NewNop()}
			ext.handleInstrumentationConfig(instrumentationConfig(tc.icName, "prod", "app"))

			res := pcommon.NewResource()
			res.Attributes().PutStr(string(semconv.K8SNamespaceNameKey), "prod")
			res.Attributes().PutStr(string(semconv.K8SContainerNameKey), "app")
			for k, v := range tc.attrs {
				res.Attributes().PutStr(k, v)
			}

			cfg, found := ext.GetFromResource(res)
			require.True(t, found, "per-source config not found for the workload's own spans")
			require.NotNil(t, cfg.PiiMasking)
			require.Equal(t, "app", cfg.ContainerName)

			require.True(t, ext.IsActiveSource(res))
		})
	}
}
