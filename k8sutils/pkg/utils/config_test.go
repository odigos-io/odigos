package utils

import (
	"context"
	"errors"
	"testing"

	"github.com/tj/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

const odigosTestNamespace = "odigos-test"

func configTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))
	return scheme
}

func effectiveConfigMap(namespace, name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       data,
	}
}

func configTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().WithScheme(configTestScheme(t)).WithObjects(objects...).Build()
}

// Every odigos controller reads the effective configuration through this function, so both the
// ConfigMap name and the key inside it are contracts with whatever renders them.
func TestEffectiveConfigMapNameAndKeyAreStable(t *testing.T) {
	assert.Equal(t, "effective-config", consts.OdigosEffectiveConfigName)
	assert.Equal(t, "config.yaml", consts.OdigosConfigurationFileName)
	assert.Equal(t, "CURRENT_NS", consts.CurrentNamespaceEnvVar)
}

func TestGetCurrentOdigosConfigurationParsesTheEffectiveConfig(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, odigosTestNamespace)

	c := configTestClient(t, effectiveConfigMap(odigosTestNamespace, consts.OdigosEffectiveConfigName,
		map[string]string{
			consts.OdigosConfigurationFileName: "" +
				"ignoredNamespaces:\n" +
				"  - kube-system\n" +
				"  - linkerd\n" +
				"ignoredContainers:\n" +
				"  - istio-proxy\n" +
				"telemetryEnabled: true\n" +
				"clusterName: staging-eu\n",
		}))

	config, err := GetCurrentOdigosConfiguration(context.Background(), c)

	assert.NoError(t, err)
	assert.Equal(t, []string{"kube-system", "linkerd"}, config.IgnoredNamespaces)
	assert.Equal(t, []string{"istio-proxy"}, config.IgnoredContainers)
	assert.True(t, config.TelemetryEnabled)
	assert.Equal(t, "staging-eu", config.ClusterName)
}

// A missing effective config is not a hard failure: the scheduler reconciles it shortly after
// odigos starts, so it gets its own sentinel that callers requeue on.
func TestGetCurrentOdigosConfigurationReportsAMissingConfigWithItsOwnSentinel(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, odigosTestNamespace)

	_, err := GetCurrentOdigosConfiguration(context.Background(), configTestClient(t))

	assert.Equal(t, ErrOdigosEffectiveConfigNotFound, err)
}

// The namespace comes from the environment rather than from an argument, so a config map with the
// right name in another namespace must not be picked up.
func TestGetCurrentOdigosConfigurationReadsTheConfigMapOfTheCurrentNamespace(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, odigosTestNamespace)

	c := configTestClient(t,
		effectiveConfigMap("some-other-namespace", consts.OdigosEffectiveConfigName,
			map[string]string{consts.OdigosConfigurationFileName: "ignoredNamespaces:\n  - from-the-wrong-namespace\n"}),
		effectiveConfigMap(odigosTestNamespace, "another-config-map",
			map[string]string{consts.OdigosConfigurationFileName: "ignoredNamespaces:\n  - from-the-wrong-config-map\n"}),
	)

	_, err := GetCurrentOdigosConfiguration(context.Background(), c)

	assert.Equal(t, ErrOdigosEffectiveConfigNotFound, err)
}

func TestGetCurrentOdigosConfigurationTreatsAnEmptyConfigMapAsAnEmptyConfiguration(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, odigosTestNamespace)

	tests := []struct {
		name string
		data map[string]string
	}{
		{name: "no data at all", data: nil},
		{name: "no configuration key", data: map[string]string{"other-key": "value"}},
		{name: "an empty configuration", data: map[string]string{consts.OdigosConfigurationFileName: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configTestClient(t, effectiveConfigMap(odigosTestNamespace,
				consts.OdigosEffectiveConfigName, tt.data))

			config, err := GetCurrentOdigosConfiguration(context.Background(), c)

			assert.NoError(t, err)
			assert.Equal(t, common.OdigosConfiguration{}, config)
		})
	}
}

func TestGetCurrentOdigosConfigurationRejectsAMalformedConfiguration(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, odigosTestNamespace)

	c := configTestClient(t, effectiveConfigMap(odigosTestNamespace, consts.OdigosEffectiveConfigName,
		map[string]string{consts.OdigosConfigurationFileName: "ignoredNamespaces: \"not-a-list\"\n"}))

	_, err := GetCurrentOdigosConfiguration(context.Background(), c)

	assert.Error(t, err)
	assert.NotEqual(t, ErrOdigosEffectiveConfigNotFound, err)
}

// Any other read failure has to stay distinguishable from "not created yet", otherwise a broken
// apiserver looks like a config that is about to appear.
func TestGetCurrentOdigosConfigurationPropagatesOtherReadErrors(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, odigosTestNamespace)

	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"},
		consts.OdigosEffectiveConfigName, errors.New("no permission to read config maps"))
	c := fake.NewClientBuilder().
		WithScheme(configTestScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, cl client.WithWatch, key client.ObjectKey,
				obj client.Object, opts ...client.GetOption) error {
				return forbidden
			},
		}).
		Build()

	_, err := GetCurrentOdigosConfiguration(context.Background(), c)

	assert.Equal(t, forbidden, err)
}

// The getter produces the sentinel and the requeue handler consumes it. Nothing links the two at
// compile time, so a wrapped or replaced error would silently turn a transient startup state into
// a reconcile failure.
func TestAMissingEffectiveConfigIsRequeuedRatherThanFailed(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, odigosTestNamespace)

	_, err := GetCurrentOdigosConfiguration(context.Background(), configTestClient(t))
	assert.Error(t, err)

	result, handlerErr := K8SNoEffectiveConfigErrorHandler(err)

	assert.NoError(t, handlerErr)
	assert.True(t, result.Requeue)
	assert.True(t, result.RequeueAfter > 0)
}

func TestIsItemIgnored(t *testing.T) {
	ignored := []string{"kube-system", "linkerd", "odigos-system"}

	tests := []struct {
		name     string
		item     string
		ignored  []string
		expected bool
	}{
		{name: "the first entry", item: "kube-system", ignored: ignored, expected: true},
		{name: "a middle entry", item: "linkerd", ignored: ignored, expected: true},
		{name: "the last entry", item: "odigos-system", ignored: ignored, expected: true},
		{name: "an item that is not ignored", item: "default", ignored: ignored, expected: false},
		{name: "matching is exact and case sensitive", item: "Kube-System", ignored: ignored, expected: false},
		{name: "matching is not a prefix match", item: "kube", ignored: ignored, expected: false},
		{name: "matching is not a substring match", item: "kube-system-extra", ignored: ignored, expected: false},
		{name: "an empty ignore list ignores nothing", item: "kube-system", ignored: nil, expected: false},
		{name: "an empty item is not ignored by default", item: "", ignored: ignored, expected: false},
		{name: "an explicitly ignored empty item", item: "", ignored: []string{""}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsItemIgnored(tt.item, tt.ignored))
		})
	}
}
