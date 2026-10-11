package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
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

	originalClient := kube.DefaultClient
	t.Cleanup(func() {
		kube.SetDefaultClient(originalClient)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effectiveConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.OdigosEffectiveConfigName,
					Namespace: env.GetCurrentNamespace(),
				},
				Data: map[string]string{
					consts.OdigosConfigurationFileName: tt.configYAML,
				},
			}
			kube.SetDefaultClient(&kube.Client{
				Interface: fake.NewSimpleClientset(effectiveConfig),
			})

			config := buildConfigResponse(context.Background(), map[string]string{
				k8sconsts.OdigosDeploymentConfigMapInstallationStatusKey: string(Finished),
			})

			assert.Equal(t, tt.want, config.InsightsEnabled)
		})
	}
}
