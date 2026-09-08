package clustercollector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	cfg "github.com/odigos-io/odigos/common/config"
)

func TestGetSecretsFromDests_ScopesManagedDestinations(t *testing.T) {
	dests := &odigosv1.DestinationList{
		Items: []odigosv1.Destination{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.otlphttp-aaaa"},
				Spec: odigosv1.DestinationSpec{
					Type: common.OtlpHttpDestinationType,
					SecretRef: &corev1.LocalObjectReference{
						Name: "secret-a",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.otlphttp-bbbb"},
				Spec: odigosv1.DestinationSpec{
					Type: common.OtlpHttpDestinationType,
					SecretRef: &corev1.LocalObjectReference{
						Name: "secret-b",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.dynamic-cccc"},
				Spec: odigosv1.DestinationSpec{
					Type: common.DynamicDestinationType,
					SecretRef: &corev1.LocalObjectReference{
						Name: "secret-c",
					},
				},
			},
		},
	}

	sources := getSecretsFromDests(dests)
	require.Len(t, sources, 3)

	assert.Equal(t, "secret-a", sources[0].SecretRef.Name)
	assert.Equal(t, cfg.DestSecretEnvPrefix("odigos.io.dest.otlphttp-aaaa"), sources[0].Prefix)

	assert.Equal(t, "secret-b", sources[1].SecretRef.Name)
	assert.Equal(t, cfg.DestSecretEnvPrefix("odigos.io.dest.otlphttp-bbbb"), sources[1].Prefix)
	assert.NotEqual(t, sources[0].Prefix, sources[1].Prefix)

	assert.Equal(t, "secret-c", sources[2].SecretRef.Name)
	assert.Empty(t, sources[2].Prefix, "dynamic destinations keep unprefixed envFrom")
}
