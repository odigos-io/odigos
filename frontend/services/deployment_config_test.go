package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildConfigResponseInsightsEnabled(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		want       bool
	}{
		{
			name:       "enabled",
			configYAML: "insights:\n  enabled: true\n",
			want:       true,
		},
		{
			name:       "disabled",
			configYAML: "insights:\n  enabled: false\n",
			want:       false,
		},
		{
			name:       "unset",
			configYAML: "configVersion: 1\n",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, configResponseFor(t, tt.configYAML).InsightsEnabled)
		})
	}
}

func TestBuildConfigResponseTraceSurgesEnabled(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		want       bool
	}{
		{
			name:       "enabled with insights",
			configYAML: "sampling:\n  traceSurge:\n    enabled: true\ninsights:\n  enabled: true\n",
			want:       true,
		},
		{
			name:       "enabled without insights",
			configYAML: "sampling:\n  traceSurge:\n    enabled: true\n",
			want:       false,
		},
		{
			name:       "enabled with insights disabled",
			configYAML: "sampling:\n  traceSurge:\n    enabled: true\ninsights:\n  enabled: false\n",
			want:       false,
		},
		{
			name:       "disabled with insights",
			configYAML: "sampling:\n  traceSurge:\n    enabled: false\ninsights:\n  enabled: true\n",
			want:       false,
		},
		{
			name:       "limits only",
			configYAML: "sampling:\n  traceSurge:\n    maxActiveSurges: 5\ninsights:\n  enabled: true\n",
			want:       false,
		},
		{
			name:       "unset",
			configYAML: "configVersion: 1\n",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, configResponseFor(t, tt.configYAML).TraceSurgesEnabled)
		})
	}
}

func configResponseFor(t *testing.T, configYAML string) model.Config {
	t.Helper()

	originalClient := kube.DefaultClient
	t.Cleanup(func() {
		kube.SetDefaultClient(originalClient)
	})

	effectiveConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: env.GetCurrentNamespace(),
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: configYAML,
		},
	}
	kube.SetDefaultClient(&kube.Client{
		Interface: fake.NewSimpleClientset(effectiveConfig),
	})

	return buildConfigResponse(context.Background(), map[string]string{
		k8sconsts.OdigosDeploymentConfigMapInstallationStatusKey: string(Finished),
	})
}
