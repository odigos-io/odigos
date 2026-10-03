package clustercollector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/metricshandler"
)

const hpaTestNamespace = "odigos-hpa-test"

// advertisedRejectionMetric returns the metric name the custom metrics API tells the aggregation
// layer about, which is the only name an HPA can successfully ask for.
func advertisedRejectionMetric(t *testing.T) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	metricshandler.DiscoveryHandler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	var discovery struct {
		Resources []struct {
			Name string `json:"name"`
		} `json:"resources"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &discovery))
	require.Len(t, discovery.Resources, 1)

	// advertised as "<resource>.<group>/<metric>"
	nameParts := strings.Split(discovery.Resources[0].Name, "/")
	require.Len(t, nameParts, 2)
	assert.Equal(t, "deployments.apps", nameParts[0],
		"the HPA describes a Deployment, so the metric has to be advertised for deployments.apps")
	return nameParts[1]
}

func hpaGatewayGroup() *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorCollectorGroupName,
			Namespace: hpaTestNamespace,
			UID:       "1f0a1f0a-0000-4000-8000-00000000cafe",
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleClusterGateway,
			ResourcesSettings: odigosv1.CollectorsGroupResourcesSettings{
				GomemlimitMiB:      400,
				CpuLimitMillicores: 1000,
			},
		},
	}
}

func hpaTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))
	require.NoError(t, apiregv1.AddToScheme(scheme))
	return scheme
}

// syncHPAFor runs the HPA sync against the given kubernetes version and returns the object it
// would have applied. The apply patch is intercepted because the fake client cannot serve one.
func syncHPAFor(t *testing.T, kubeVersion string, gateway *odigosv1.CollectorsGroup, objects []client.Object, patchErr error) (client.Object, error) {
	t.Helper()

	previousConfig := commonconfig.ControllerConfig
	t.Cleanup(func() { commonconfig.ControllerConfig = previousConfig })
	commonconfig.ControllerConfig = &controllerconfig.ControllerConfig{
		K8sVersion: version.MustParseSemantic(kubeVersion),
	}

	var applied client.Object
	scheme := hpaTestScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).
		WithInterceptorFuncs(interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				assert.Equal(t, client.Apply.Type(), patch.Type())
				applied = obj
				return patchErr
			},
		}).Build()

	err := syncHPA(gateway, logr.NewContext(context.Background(), logr.Discard()), k8sClient, scheme)
	return applied, err
}

func applyHPA(t *testing.T, kubeVersion string, gateway *odigosv1.CollectorsGroup, objects []client.Object) client.Object {
	t.Helper()

	applied, err := syncHPAFor(t, kubeVersion, gateway, objects, nil)
	require.NoError(t, err)
	require.NotNil(t, applied)
	return applied
}

func odigosCustomMetricsAPIService() *apiregv1.APIService {
	return &apiregv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.CustomMetricsAPIServiceName},
		Spec: apiregv1.APIServiceSpec{
			Service: &apiregv1.ServiceReference{
				Name:      k8sconsts.AutoScalerWebhookServiceName,
				Namespace: hpaTestNamespace,
			},
		},
	}
}

// ****************
// buildv2Metrics / buildv2beta2Metrics
// ****************

func TestBuildv2Metrics_WithoutTheCustomMetric(t *testing.T) {
	metrics := buildv2Metrics(false, resource.MustParse("300Mi"), resource.MustParse("750m"))

	require.Len(t, metrics, 2)
	// without the odigos custom metrics API the HPA must fall back to resource metrics only; an
	// object metric the metrics server cannot answer makes the whole HPA unable to scale.
	for _, metric := range metrics {
		assert.Equal(t, autoscalingv2.ResourceMetricSourceType, metric.Type)
		assert.Nil(t, metric.Object)
	}
	assert.Equal(t, corev1.ResourceMemory, metrics[0].Resource.Name)
	assert.Equal(t, resource.MustParse("300Mi"), *metrics[0].Resource.Target.AverageValue)
	assert.Equal(t, corev1.ResourceCPU, metrics[1].Resource.Name)
	assert.Equal(t, resource.MustParse("750m"), *metrics[1].Resource.Target.AverageValue)
}

func TestBuildv2Metrics_WithTheCustomMetric(t *testing.T) {
	metrics := buildv2Metrics(true, resource.MustParse("300Mi"), resource.MustParse("750m"))

	require.Len(t, metrics, 3)
	rejections := metrics[0]
	require.Equal(t, autoscalingv2.ObjectMetricSourceType, rejections.Type)
	require.NotNil(t, rejections.Object)
	assert.Equal(t, advertisedRejectionMetric(t), rejections.Object.Metric.Name)
	assert.Equal(t, "odigos-gateway", rejections.Object.DescribedObject.Name)
	assert.Equal(t, "Deployment", rejections.Object.DescribedObject.Kind)
	assert.Equal(t, "apps/v1", rejections.Object.DescribedObject.APIVersion)
	assert.Equal(t, autoscalingv2.ValueMetricType, rejections.Object.Target.Type)

	// the custom metric is binary: 1 when at least half the gateway pods reject data, 0 otherwise.
	// The target has to sit strictly between the two, or the signal either never fires or fires
	// permanently.
	target := rejections.Object.Target.Value.AsApproximateFloat64()
	assert.Greater(t, target, 0.0)
	assert.Less(t, target, 1.0)

	assert.Equal(t, autoscalingv2.ResourceMetricSourceType, metrics[1].Type)
	assert.Equal(t, autoscalingv2.ResourceMetricSourceType, metrics[2].Type)
}

func TestBuildv2beta2Metrics_WithoutTheCustomMetric(t *testing.T) {
	metrics := buildv2beta2Metrics(false, resource.MustParse("300Mi"), resource.MustParse("750m"))

	require.Len(t, metrics, 2)
	for _, metric := range metrics {
		assert.Equal(t, autoscalingv2beta2.ResourceMetricSourceType, metric.Type)
		assert.Nil(t, metric.Object)
	}
	assert.Equal(t, corev1.ResourceMemory, metrics[0].Resource.Name)
	assert.Equal(t, resource.MustParse("300Mi"), *metrics[0].Resource.Target.AverageValue)
	assert.Equal(t, corev1.ResourceCPU, metrics[1].Resource.Name)
	assert.Equal(t, resource.MustParse("750m"), *metrics[1].Resource.Target.AverageValue)
}

func TestBuildv2beta2Metrics_WithTheCustomMetric(t *testing.T) {
	metrics := buildv2beta2Metrics(true, resource.MustParse("300Mi"), resource.MustParse("750m"))

	require.Len(t, metrics, 3)
	rejections := metrics[0]
	require.Equal(t, autoscalingv2beta2.ObjectMetricSourceType, rejections.Type)
	require.NotNil(t, rejections.Object)
	assert.Equal(t, advertisedRejectionMetric(t), rejections.Object.Metric.Name)
	assert.Equal(t, "odigos-gateway", rejections.Object.DescribedObject.Name)
	assert.Equal(t, "Deployment", rejections.Object.DescribedObject.Kind)
	assert.Equal(t, "apps/v1", rejections.Object.DescribedObject.APIVersion)
	assert.Equal(t, autoscalingv2beta2.ValueMetricType, rejections.Object.Target.Type)

	target := rejections.Object.Target.Value.AsApproximateFloat64()
	assert.Greater(t, target, 0.0)
	assert.Less(t, target, 1.0)
}

// ****************
// syncHPA
// ****************

func TestSyncHPA_ModernClusterUsesTheStableAPI(t *testing.T) {
	// autoscaling/v2 is stable from 1.25 and v2beta2 is gone in 1.26, so 1.25 has to be on this
	// side of the switch
	for _, kubeVersion := range []string{"1.25.0", "1.31.0"} {
		t.Run(kubeVersion, func(t *testing.T) {
			assertStableHPA(t, applyHPA(t, kubeVersion, hpaGatewayGroup(), nil))
		})
	}
}

func assertStableHPA(t *testing.T, applied client.Object) {
	t.Helper()

	hpa, ok := applied.(*autoscalingv2.HorizontalPodAutoscaler)
	require.Truef(t, ok, "expected an autoscaling/v2 HPA, got %T", applied)
	assert.Equal(t, "autoscaling/v2", hpa.APIVersion)
	assert.Equal(t, "HorizontalPodAutoscaler", hpa.Kind)
	assert.Equal(t, k8sconsts.OdigosClusterCollectorHpaName, hpa.Name)
	assert.Equal(t, hpaTestNamespace, hpa.Namespace)
	assert.Equal(t, "odigos-gateway", hpa.Spec.ScaleTargetRef.Name)
	assert.Equal(t, "Deployment", hpa.Spec.ScaleTargetRef.Kind)
	require.NotNil(t, hpa.Spec.MinReplicas)
	assert.Equal(t, int32(1), *hpa.Spec.MinReplicas)
	assert.Equal(t, int32(10), hpa.Spec.MaxReplicas)

	// 75% of the configured limits
	assert.Equal(t, resource.MustParse("300Mi"), *hpa.Spec.Metrics[0].Resource.Target.AverageValue)
	assert.Equal(t, resource.MustParse("750m"), *hpa.Spec.Metrics[1].Resource.Target.AverageValue)

	// scaling out has to be immediate and scaling in gradual, or a gateway under memory pressure
	// oscillates instead of absorbing the load
	require.NotNil(t, hpa.Spec.Behavior)
	assert.Equal(t, int32(0), *hpa.Spec.Behavior.ScaleUp.StabilizationWindowSeconds)
	assert.Equal(t, autoscalingv2.MaxChangePolicySelect, *hpa.Spec.Behavior.ScaleUp.SelectPolicy)
	assert.Equal(t, int32(900), *hpa.Spec.Behavior.ScaleDown.StabilizationWindowSeconds)
	assert.Equal(t, autoscalingv2.MinChangePolicySelect, *hpa.Spec.Behavior.ScaleDown.SelectPolicy)

	// the HPA is only garbage collected with the collectors group if the group owns it as its
	// controller; a leftover HPA keeps scaling a deployment odigos no longer manages
	require.Len(t, hpa.OwnerReferences, 1)
	assert.Equal(t, "CollectorsGroup", hpa.OwnerReferences[0].Kind)
	assert.Equal(t, k8sconsts.OdigosClusterCollectorCollectorGroupName, hpa.OwnerReferences[0].Name)
	require.NotNil(t, hpa.OwnerReferences[0].Controller)
	assert.True(t, *hpa.OwnerReferences[0].Controller)
}

func TestSyncHPA_MidRangeClusterUsesV2beta2(t *testing.T) {
	for _, kubeVersion := range []string{"1.23.0", "1.24.7"} {
		t.Run(kubeVersion, func(t *testing.T) {
			applied := applyHPA(t, kubeVersion, hpaGatewayGroup(), nil)

			hpa, ok := applied.(*autoscalingv2beta2.HorizontalPodAutoscaler)
			require.Truef(t, ok, "expected an autoscaling/v2beta2 HPA, got %T", applied)
			assert.Equal(t, "autoscaling/v2beta2", hpa.APIVersion)
			require.NotNil(t, hpa.Spec.Behavior)
			assert.Equal(t, autoscalingv2beta2.ScalingPolicySelect("Max"), *hpa.Spec.Behavior.ScaleUp.SelectPolicy)
			assert.Equal(t, autoscalingv2beta2.ScalingPolicySelect("Min"), *hpa.Spec.Behavior.ScaleDown.SelectPolicy)
			assert.Len(t, hpa.Spec.Metrics, 2)
		})
	}
}

func TestSyncHPA_LegacyClusterUsesV2beta1WithoutBehavior(t *testing.T) {
	applied := applyHPA(t, "1.22.9", hpaGatewayGroup(), nil)

	hpa, ok := applied.(*autoscalingv2beta1.HorizontalPodAutoscaler)
	require.Truef(t, ok, "expected an autoscaling/v2beta1 HPA, got %T", applied)
	assert.Equal(t, "autoscaling/v2beta1", hpa.APIVersion)
	// v2beta1 has neither behavior nor object metrics
	require.Len(t, hpa.Spec.Metrics, 2)
	assert.Equal(t, corev1.ResourceMemory, hpa.Spec.Metrics[0].Resource.Name)
	assert.Equal(t, resource.MustParse("300Mi"), *hpa.Spec.Metrics[0].Resource.TargetAverageValue)
	assert.Equal(t, corev1.ResourceCPU, hpa.Spec.Metrics[1].Resource.Name)
	assert.Equal(t, resource.MustParse("750m"), *hpa.Spec.Metrics[1].Resource.TargetAverageValue)
}

func TestSyncHPA_RejectionMetricIsUsedOnlyWhenOdigosServesIt(t *testing.T) {
	rejectionMetric := advertisedRejectionMetric(t)

	objectMetricNames := func(applied client.Object) []string {
		t.Helper()

		hpa, ok := applied.(*autoscalingv2.HorizontalPodAutoscaler)
		require.True(t, ok)
		names := []string{}
		for _, metric := range hpa.Spec.Metrics {
			if metric.Object != nil {
				names = append(names, metric.Object.Metric.Name)
			}
		}
		return names
	}

	t.Run("odigos serves the custom metrics API", func(t *testing.T) {
		applied := applyHPA(t, "1.31.0", hpaGatewayGroup(),
			[]client.Object{odigosCustomMetricsAPIService()})

		assert.Equal(t, []string{rejectionMetric}, objectMetricNames(applied))
	})

	t.Run("another adapter serves the custom metrics API", func(t *testing.T) {
		// asking a foreign adapter for odigos_gateway_rejections leaves the HPA with a metric it
		// can never read, which blocks it from acting on the CPU and memory metrics as well
		foreign := odigosCustomMetricsAPIService()
		foreign.Spec.Service.Name = "prometheus-adapter"

		applied := applyHPA(t, "1.31.0", hpaGatewayGroup(), []client.Object{foreign})

		assert.Empty(t, objectMetricNames(applied))
	})

	t.Run("no custom metrics API is registered", func(t *testing.T) {
		applied := applyHPA(t, "1.31.0", hpaGatewayGroup(), nil)

		assert.Empty(t, objectMetricNames(applied))
	})
}

func TestSyncHPA_ReplicaBounds(t *testing.T) {
	replicas := func(n int) *int { return &n }

	for _, tc := range []struct {
		name        string
		minReplicas *int
		maxReplicas *int
		expectedMin int32
		expectedMax int32
	}{
		{
			name:        "unset falls back to the defaults",
			expectedMin: 1,
			expectedMax: 10,
		},
		{
			name:        "configured bounds are used",
			minReplicas: replicas(3),
			maxReplicas: replicas(25),
			expectedMin: 3,
			expectedMax: 25,
		},
		{
			// scaling the gateway to zero replicas would drop all telemetry, so a
			// non-positive configuration must not be honoured
			name:        "non positive bounds fall back to the defaults",
			minReplicas: replicas(0),
			maxReplicas: replicas(0),
			expectedMin: 1,
			expectedMax: 10,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gateway := hpaGatewayGroup()
			gateway.Spec.ResourcesSettings.MinReplicas = tc.minReplicas
			gateway.Spec.ResourcesSettings.MaxReplicas = tc.maxReplicas

			applied := applyHPA(t, "1.31.0", gateway, nil)

			hpa, ok := applied.(*autoscalingv2.HorizontalPodAutoscaler)
			require.True(t, ok)
			require.NotNil(t, hpa.Spec.MinReplicas)
			assert.Equal(t, tc.expectedMin, *hpa.Spec.MinReplicas)
			assert.Equal(t, tc.expectedMax, hpa.Spec.MaxReplicas)
		})
	}
}

func TestSyncHPA_TargetsARenamedGatewayDeployment(t *testing.T) {
	gateway := hpaGatewayGroup()
	gateway.Spec.DeploymentName = "custom-gateway-name"

	applied := applyHPA(t, "1.31.0", gateway, nil)

	hpa, ok := applied.(*autoscalingv2.HorizontalPodAutoscaler)
	require.True(t, ok)
	assert.Equal(t, "custom-gateway-name", hpa.Spec.ScaleTargetRef.Name)
}

func TestSyncHPA_ApplyFailureIsReported(t *testing.T) {
	applyErr := errors.New("hpa is forbidden")

	_, err := syncHPAFor(t, "1.31.0", hpaGatewayGroup(), nil, applyErr)

	require.ErrorIs(t, err, applyErr)
}
