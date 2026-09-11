package insights

import (
	"context"
	"fmt"
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func effectiveConfigMap(configYAML string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: env.GetCurrentNamespace(),
		},
		Data: map[string]string{consts.OdigosConfigurationFileName: configYAML},
	}
}

func newClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		objects []client.Object
		want    bool
	}{
		{
			name:    "no effective config at all",
			objects: nil,
			want:    false,
		},
		{
			name:    "insights key absent from config",
			objects: []client.Object{effectiveConfigMap("configVersion: 1\n")},
			want:    false,
		},
		{
			name:    "insights present but enabled unset",
			objects: []client.Object{effectiveConfigMap("insights: {}\n")},
			want:    false,
		},
		{
			name:    "insights explicitly disabled",
			objects: []client.Object{effectiveConfigMap("insights:\n  enabled: false\n")},
			want:    false,
		},
		{
			name:    "insights enabled",
			objects: []client.Object{effectiveConfigMap("insights:\n  enabled: true\n")},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsEnabled(context.Background(), newClient(tt.objects...))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnabledFromOdigosConfigNilSafe(t *testing.T) {
	assert.False(t, EnabledFromOdigosConfig(nil))
}

func TestErrNotEnabledIsMatchableWhenWrapped(t *testing.T) {
	wrapped := fmt.Errorf("listing findings: %w", ErrNotEnabled)
	assert.ErrorIs(t, wrapped, ErrNotEnabled)
}

func TestIsEnabledPropagatesAConfigParseFailure(t *testing.T) {
	_, err := IsEnabled(context.Background(), newClient(effectiveConfigMap("insights: [not-a-map")))
	require.Error(t, err)
}
