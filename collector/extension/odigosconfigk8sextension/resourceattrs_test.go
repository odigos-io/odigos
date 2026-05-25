package odigosconfigk8sextension

import (
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestWorkloadContainerKeyPrefersOdigosWorkloadIdentity(t *testing.T) {
	tests := []struct {
		name      string
		attrs     map[string]string
		wantKey   string
		wantError bool
	}{
		{
			name: "cronjob profiles enriched with job owner",
			attrs: map[string]string{
				string(semconv.K8SNamespaceNameKey): "jobs",
				string(semconv.K8SJobNameKey):       "nightly-28472910",
				string(semconv.K8SCronJobNameKey):   "nightly",
				consts.OdigosWorkloadKindAttribute:  "CronJob",
				consts.OdigosWorkloadNameAttribute:  "nightly",
			},
			wantKey: "jobs/CronJob/nightly/",
		},
		{
			name: "deploymentconfig profiles enriched as deployment",
			attrs: map[string]string{
				string(semconv.K8SNamespaceNameKey):  "openshift",
				string(semconv.K8SDeploymentNameKey): "backend",
				consts.OdigosWorkloadKindAttribute:   "DeploymentConfig",
				consts.OdigosWorkloadNameAttribute:   "backend",
			},
			wantKey: "openshift/DeploymentConfig/backend/",
		},
		{
			name: "non odigos telemetry falls back to k8s semconv",
			attrs: map[string]string{
				string(semconv.K8SNamespaceNameKey):  "default",
				string(semconv.K8SDeploymentNameKey): "frontend",
			},
			wantKey: "default/Deployment/frontend/",
		},
		{
			name: "partial odigos identity still falls back to k8s semconv",
			attrs: map[string]string{
				string(semconv.K8SNamespaceNameKey): "jobs",
				string(semconv.K8SJobNameKey):       "batch",
				consts.OdigosWorkloadKindAttribute:  "CronJob",
			},
			wantKey: "jobs/Job/batch/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := workloadContainerKeyFromResourceAttributes(resourceAttributes(tt.attrs))
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantKey, key)
		})
	}
}

func TestWorkloadKeyPrefersOdigosWorkloadIdentity(t *testing.T) {
	attrs := resourceAttributes(map[string]string{
		string(semconv.K8SNamespaceNameKey): "jobs",
		string(semconv.K8SContainerNameKey): "worker",
		string(semconv.K8SJobNameKey):       "nightly-28472910",
		string(semconv.K8SCronJobNameKey):   "nightly",
		consts.OdigosWorkloadKindAttribute:  "CronJob",
		consts.OdigosWorkloadNameAttribute:  "nightly",
	})

	key, err := workloadKeyFromResourceAttributes(attrs)
	require.NoError(t, err)
	require.Equal(t, "jobs/CronJob/nightly/worker", key)
}

func resourceAttributes(attrs map[string]string) pcommon.Map {
	m := pcommon.NewMap()
	for key, value := range attrs {
		m.PutStr(key, value)
	}
	return m
}
