package clustercollector

import (
	"context"
	"slices"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

// The odigos_enterprise_auth extension and the profiling wiring are only available in the
// enterprise collector image, and ODIGOS_ONPREM_TOKEN is only available from the odigos-pro secret
// on enterprise tier. Rendering either of them for a community cluster crashloops the collector,
// and omitting them on enterprise tier makes the auth extension refuse to start, so both
// directions of these gates are asserted here.

// ****************
// Setup helpers
// ****************

func newTierGateScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	return scheme
}

// ControllerConfig is a package level global that the envtest suite in this package also sets,
// so restore it to keep the tests order independent.
func setTierGateControllerConfig(t *testing.T) {
	t.Helper()

	previous := commonconfig.ControllerConfig
	t.Cleanup(func() { commonconfig.ControllerConfig = previous })
	commonconfig.ControllerConfig = &controllerconfig.ControllerConfig{
		K8sVersion:     version.MustParseSemantic("0.0.0"),
		CollectorImage: "otelcol",
	}
}

func newTierGateGateway() *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorConfigMapName,
			Namespace: odigosconsts.DefaultOdigosNamespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleClusterGateway,
			ResourcesSettings: odigosv1.CollectorsGroupResourcesSettings{
				MemoryRequestMiB:     256,
				MemoryLimitMiB:       512,
				CpuRequestMillicores: 250,
				CpuLimitMillicores:   500,
				GomemlimitMiB:        200,
			},
			CollectorOwnMetricsPort: 8888,
		},
	}
}

func newTierGateDestination() *odigosv1.Destination {
	return &odigosv1.Destination{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-destination",
			Namespace: odigosconsts.DefaultOdigosNamespace,
		},
		Spec: odigosv1.DestinationSpec{
			Type:            odigoscommon.OtlpHttpDestinationType,
			DestinationName: "test-destination",
			Data:            map[string]string{"OTLP_HTTP_ENDPOINT": "http://test-endpoint:4318"},
			Signals:         []odigoscommon.ObservabilitySignal{odigoscommon.TracesObservabilitySignal},
		},
	}
}

func newTierGateClient(t *testing.T, scheme *runtime.Scheme, odigosConfig odigoscommon.OdigosConfiguration, objects ...client.Object) client.Client {
	t.Helper()

	rawConfig, err := yaml.Marshal(odigosConfig)
	require.NoError(t, err)

	effectiveConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      odigosconsts.OdigosEffectiveConfigName,
			Namespace: odigosconsts.DefaultOdigosNamespace,
		},
		Data: map[string]string{odigosconsts.OdigosConfigurationFileName: string(rawConfig)},
	}

	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(effectiveConfig)
	if len(objects) > 0 {
		builder = builder.WithObjects(objects...).WithStatusSubresource(objects...)
	}
	return builder.Build()
}

func readRenderedGatewayConfig(t *testing.T, c client.Client) config.Config {
	t.Helper()

	var cm corev1.ConfigMap
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{
		Namespace: odigosconsts.DefaultOdigosNamespace,
		Name:      k8sconsts.OdigosClusterCollectorConfigMapName,
	}, &cm))

	var collectorConfig config.Config
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data[k8sconsts.OdigosClusterCollectorConfigMapKey]), &collectorConfig))
	return collectorConfig
}

func findEnvVar(container corev1.Container, name string) *corev1.EnvVar {
	for i := range container.Env {
		if container.Env[i].Name == name {
			return &container.Env[i]
		}
	}
	return nil
}

// ****************
// Tests
// ****************

func TestSyncConfigMap_EnterpriseComponentsGatedByTier(t *testing.T) {
	setTierGateControllerConfig(t)

	for _, tc := range []struct {
		name           string
		tier           odigoscommon.OdigosTier
		wantEnterprise bool
	}{
		{"community renders no enterprise components", odigoscommon.CommunityOdigosTier, false},
		{"onprem renders the enterprise components", odigoscommon.OnPremOdigosTier, true},
		{"cloud renders the enterprise components", odigoscommon.CloudOdigosTier, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := newTierGateScheme(t)
			dest := newTierGateDestination()
			profilingEnabled := true
			c := newTierGateClient(t, scheme, odigoscommon.OdigosConfiguration{
				Profiling: &odigoscommon.ProfilingConfiguration{Enabled: &profilingEnabled},
			}, dest)

			signals, err := syncConfigMap(&odigosv1.DestinationList{Items: []odigosv1.Destination{*dest}},
				&odigosv1.ProcessorList{}, newTierGateGateway(), context.Background(), c, scheme, tc.tier)
			require.NoError(t, err)
			require.Contains(t, signals, odigoscommon.TracesObservabilitySignal,
				"precondition: the destination pipeline should be rendered regardless of tier")

			collectorConfig := readRenderedGatewayConfig(t, c)

			_, authDefined := collectorConfig.Extensions[odigosEnterpriseAuthExtensionName]
			assert.Equal(t, tc.wantEnterprise, authDefined, "extensions map: %v", collectorConfig.Extensions)
			assert.Equal(t, tc.wantEnterprise,
				slices.Contains(collectorConfig.Service.Extensions, odigosEnterpriseAuthExtensionName),
				"service.extensions: %v", collectorConfig.Service.Extensions)

			_, profilesWired := collectorConfig.Service.Pipelines["profiles"]
			assert.Equal(t, tc.wantEnterprise, profilesWired,
				"profiles pipeline presence should follow tier, pipelines: %v", collectorConfig.Service.Pipelines)

			// health_check and pprof must survive in both tiers.
			assert.Contains(t, collectorConfig.Service.Extensions, "health_check")
			assert.Contains(t, collectorConfig.Service.Extensions, "pprof")
		})
	}
}

func TestGetDesiredDeployment_OnpremTokenGatedByTier(t *testing.T) {
	setTierGateControllerConfig(t)

	for _, tc := range []struct {
		name      string
		tier      odigoscommon.OdigosTier
		wantToken bool
	}{
		{"community omits the onprem token", odigoscommon.CommunityOdigosTier, false},
		{"onprem injects the onprem token", odigoscommon.OnPremOdigosTier, true},
		{"cloud injects the onprem token", odigoscommon.CloudOdigosTier, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := newTierGateScheme(t)
			c := newTierGateClient(t, scheme, odigoscommon.OdigosConfiguration{})

			dep, err := getDesiredDeployment(context.Background(), c, &odigosv1.DestinationList{},
				"config-hash", newTierGateGateway(), scheme, "v1.2.3", nil, tc.tier)
			require.NoError(t, err)
			require.Len(t, dep.Spec.Template.Spec.Containers, 1)

			tokenEnv := findEnvVar(dep.Spec.Template.Spec.Containers[0], k8sconsts.OdigosOnpremTokenEnvName)
			if !tc.wantToken {
				assert.Nil(t, tokenEnv, "%s must not be set on community tier, where the odigos-pro secret does not exist",
					k8sconsts.OdigosOnpremTokenEnvName)
				return
			}

			require.NotNil(t, tokenEnv, "%s should be set on tier %q", k8sconsts.OdigosOnpremTokenEnvName, tc.tier)
			assert.Empty(t, tokenEnv.Value, "the token must not be rendered into the manifest")
			require.NotNil(t, tokenEnv.ValueFrom)
			require.NotNil(t, tokenEnv.ValueFrom.SecretKeyRef, "the token must be read from a secret")
			assert.Equal(t, k8sconsts.OdigosProSecretName, tokenEnv.ValueFrom.SecretKeyRef.Name)
			assert.Equal(t, k8sconsts.OdigosOnpremTokenSecretKey, tokenEnv.ValueFrom.SecretKeyRef.Key)
		})
	}
}

// The onprem token is appended to the same env var slice as the proxy settings, so neither may
// drop the other.
func TestGetDesiredDeployment_OnpremTokenCoexistsWithHttpsProxy(t *testing.T) {
	setTierGateControllerConfig(t)

	scheme := newTierGateScheme(t)
	gateway := newTierGateGateway()
	proxyAddress := "http://my-proxy:8080"
	gateway.Spec.HttpsProxyAddress = &proxyAddress

	dep, err := getDesiredDeployment(context.Background(), newTierGateClient(t, scheme, odigoscommon.OdigosConfiguration{}),
		&odigosv1.DestinationList{}, "config-hash", gateway, scheme, "v1.2.3", nil, odigoscommon.OnPremOdigosTier)
	require.NoError(t, err)

	container := dep.Spec.Template.Spec.Containers[0]
	assert.NotNil(t, findEnvVar(container, k8sconsts.OdigosOnpremTokenEnvName))

	httpsProxy := findEnvVar(container, "HTTPS_PROXY")
	require.NotNil(t, httpsProxy)
	assert.Equal(t, proxyAddress, httpsProxy.Value)
	assert.NotNil(t, findEnvVar(container, "NO_PROXY"))
}
