package diagnose

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"
)

func dgConfigMap(name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: dgNamespace, Name: name, ManagedFields: dgManagedFields()},
		Data:       data,
	}
}

func TestFetchConfigMapsCollectsEveryConfigMapInTheOdigosNamespace(t *testing.T) {
	client := dgClientset(
		dgConfigMap("odigos-config", map[string]string{"config.yaml": "telemetryEnabled: true"}),
		dgConfigMap("odigos-own-telemetry-otelcol-conf", map[string]string{"conf": "receivers: {}"}),
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: dgAppNs, Name: "app-config"}},
	)
	builder := newDgBuilder()
	dir := GetConfigMapsDir(dgRootDir, dgNamespace)

	require.NoError(t, FetchConfigMaps(context.Background(), client, builder, dir, dgNamespace))

	assert.Equal(t, []string{
		dir + "/configmap-odigos-config.yaml",
		dir + "/configmap-odigos-own-telemetry-otelcol-conf.yaml",
	}, builder.paths(), "only the odigos namespace is collected")

	var collected corev1.ConfigMap
	require.NoError(t, yaml.Unmarshal(builder.body(t, dir+"/configmap-odigos-config.yaml"), &collected))
	assert.Equal(t, "telemetryEnabled: true", collected.Data["config.yaml"])
	assert.Empty(t, collected.ManagedFields)
}

func TestFetchConfigMapsReportsWhyItCouldNotList(t *testing.T) {
	client := dgClientset()
	client.PrependReactor("list", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "", fmt.Errorf("denied"))
	})

	err := FetchConfigMaps(context.Background(), client, newDgBuilder(), "dir", dgNamespace)

	assert.ErrorContains(t, err, "failed to list configmaps")
	assert.True(t, apierrors.IsForbidden(err), "the reason must survive wrapping")
}

func TestFetchConfigMapsKeepsGoingWhenOneConfigMapCannotBeWritten(t *testing.T) {
	client := dgClientset(dgConfigMap("odigos-config", nil), dgConfigMap("odigos-tier", nil))
	builder := newDgBuilder()
	builder.failOn = func(_, filename string) error {
		if filename == "configmap-odigos-config.yaml" {
			return fmt.Errorf("no space left on device")
		}
		return nil
	}

	require.NoError(t, FetchConfigMaps(context.Background(), client, builder, "dir", dgNamespace))

	assert.Equal(t, []string{"dir/configmap-odigos-tier.yaml"}, builder.paths())
}
