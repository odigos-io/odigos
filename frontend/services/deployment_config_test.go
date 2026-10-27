package services

import (
	"context"
	"errors"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
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

func TestIsReadonlyMode(t *testing.T) {
	originalClient := kube.DefaultClient
	t.Cleanup(func() {
		kube.SetDefaultClient(originalClient)
	})

	configMap := func(yaml string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OdigosEffectiveConfigName,
				Namespace: env.GetCurrentNamespace(),
			},
			Data: map[string]string{consts.OdigosConfigurationFileName: yaml},
		}
	}

	tests := []struct {
		name   string
		client func() kubernetes.Interface
		want   bool
	}{
		{
			name:   "readonly",
			client: func() kubernetes.Interface { return fake.NewSimpleClientset(configMap("uiMode: readonly\n")) },
			want:   true,
		},
		{
			name:   "default mode",
			client: func() kubernetes.Interface { return fake.NewSimpleClientset(configMap("uiMode: default\n")) },
			want:   false,
		},
		{
			name:   "config map missing",
			client: func() kubernetes.Interface { return fake.NewSimpleClientset() },
			want:   true,
		},
		{
			name:   "config unparsable",
			client: func() kubernetes.Interface { return fake.NewSimpleClientset(configMap("uiMode: [unclosed")) },
			want:   true,
		},
		{
			name: "api error",
			client: func() kubernetes.Interface {
				cs := fake.NewSimpleClientset()
				cs.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("api unavailable")
				})
				return cs
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube.SetDefaultClient(&kube.Client{Interface: tt.client()})
			assert.Equal(t, tt.want, IsReadonlyMode(context.Background()))
		})
	}
}
