package nodecollector

import (
	"context"
	"os"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
)

const (
	mockNamespaceBase   = "test-namespace"
	mockDeploymentName  = "test-deployment"
	mockDaemonSetName   = "test-daemonset"
	mockStatefulSetName = "test-statefulset"
)

func NewMockNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewMockTestDeployment(ns *corev1.Namespace) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDeploymentName,
			Namespace: ns.GetName(),
		},
	}
}

func NewMockTestDaemonSet(ns *corev1.Namespace) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDaemonSetName,
			Namespace: ns.GetName(),
		},
	}
}

func NewMockTestStatefulSet(ns *corev1.Namespace) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockStatefulSetName,
			Namespace: ns.GetName(),
		},
	}
}

// givin a workload object (deployment, daemonset, statefulset) return a mock instrumented application
// with a single container with the GoProgrammingLanguage
func NewMockInstrumentationConfig(workloadObject client.Object) *odigosv1.InstrumentationConfig {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName(workloadObject.GetName(), gvk.Kind),
			Namespace: workloadObject.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: workloadObject.GetName(),
					Kind: gvk.Kind,
				},
			},
		},
	}
}

func NewMockInstrumentationConfigWoOwner(workloadObject client.Object) *odigosv1.InstrumentationConfig {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName(workloadObject.GetName(), gvk.Kind),
			Namespace: workloadObject.GetNamespace(),
		},
	}
}

// Destination list must include a destination with LogsObservabilitySignal for the filelog to be configured
func NewMockDestinationList() *odigosv1.DestinationList {
	return &odigosv1.DestinationList{
		Items: []v1alpha1.Destination{
			{
				Spec: v1alpha1.DestinationSpec{
					Signals: []common.ObservabilitySignal{
						common.LogsObservabilitySignal,
					},
				},
			},
		},
	}
}

func openTestData(t *testing.T, path string) string {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to open %s", path)
	}
	return string(want)
}

func TestCalculateConfigMapData(t *testing.T) {
	want := openTestData(t, "testdata/logs_included.yaml")

	ns := NewMockNamespace("default")
	ns2 := NewMockNamespace("other-namespace")

	items := []v1alpha1.InstrumentationConfig{
		*NewMockInstrumentationConfig(NewMockTestDeployment(ns)),
		*NewMockInstrumentationConfig(NewMockTestDaemonSet(ns)),
		*NewMockInstrumentationConfig(NewMockTestStatefulSet(ns2)),
		*NewMockInstrumentationConfigWoOwner(NewMockTestDeployment(ns2)),
	}

	trueVal := true
	falseVal := false

	_, got, err := calculateCollectorConfigDomains(
		context.Background(),
		"odigos-system",
		&odigosv1.CollectorsGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "test-collector-group"},
			Spec: odigosv1.CollectorsGroupSpec{
				CollectorOwnMetricsPort: 4317,
				ResourceDetectors: &common.ResourceDetectorsConfiguration{
					EC2:   &common.ResourceDetectorConfig{Enabled: &trueVal},
					EKS:   &common.ResourceDetectorConfig{Enabled: &falseVal},
					Azure: &common.ResourceDetectorConfig{Enabled: &trueVal},
					AKS:   &common.ResourceDetectorConfig{Enabled: &trueVal},
				},
				Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{
					HostMetrics: &common.MetricsSourceHostMetricsConfiguration{
						Interval: "33s",
					},
					KubeletStats: &common.MetricsSourceKubeletStatsConfiguration{
						Interval: "44s",
					},
					AgentsTelemetry: &odigosv1.AgentsTelemetrySettings{},
				},
			},
		},
		&odigosv1.InstrumentationConfigList{
			Items: items,
		},
		[]common.ObservabilitySignal{
			common.LogsObservabilitySignal,
			common.MetricsObservabilitySignal,
			common.TracesObservabilitySignal,
		},
		[]*v1alpha1.Processor{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "test_processor"},
				Spec: v1alpha1.ProcessorSpec{
					OrderHint:      1,
					Type:           "test_type",
					CollectorRoles: []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleNodeCollector},
					Disabled:       false,
					ProcessorConfig: runtime.RawExtension{
						Raw: []byte(`{"key":"val"}`),
					},
					Signals: []common.ObservabilitySignal{
						common.LogsObservabilitySignal,
						common.MetricsObservabilitySignal,
						common.TracesObservabilitySignal,
					},
				},
			},
		},
		false,                   /* onGKE */
		true,                    /* loadBalancingNeeded */
		nil,                     /* profiling */
		common.OnPremOdigosTier, /* tier */
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, want, got)
}

// The own-kubelet pipeline reuses the destination's kubeletstats receiver and enriches its
// metrics with k8sattributes. Assert on the rendered config that the receiver it references
// really exists, and that its informers are scoped - the node collector is a DaemonSet, so an
// unscoped informer is a cluster-wide pod LIST/WATCH from every node.
func TestCalculateConfigMapDataOwnKubeletMetrics(t *testing.T) {
	configDomains, mergedYaml, err := calculateCollectorConfigDomains(
		context.Background(),
		"odigos-system",
		&odigosv1.CollectorsGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "test-collector-group"},
			Spec: odigosv1.CollectorsGroupSpec{
				CollectorOwnMetricsPort: 4317,
				Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{
					KubeletStats: &common.MetricsSourceKubeletStatsConfiguration{
						Interval: "10s",
					},
					OdigosOwnMetrics: &odigosv1.OdigosOwnMetricsSettings{Interval: "10s"},
				},
			},
		},
		&odigosv1.InstrumentationConfigList{},
		[]common.ObservabilitySignal{common.MetricsObservabilitySignal},
		nil,
		false,                   /* onGKE */
		false,                   /* loadBalancingNeeded */
		nil,                     /* profiling */
		common.OnPremOdigosTier, /* tier */
	)
	assert.NoError(t, err)

	odigletMetrics, ok := configDomains["odiglet_metrics"]
	assert.True(t, ok, "odiglet_metrics domain should be configured")

	pipeline, ok := odigletMetrics.Service.Pipelines["metrics/own-kubelet"]
	assert.True(t, ok, "own-kubelet pipeline should be configured")
	assert.Equal(t, []string{"kubeletstats"}, pipeline.Receivers)

	// the receiver is owned by the metrics domain, so it must survive the merge
	mergedConfig := config.Config{}
	assert.NoError(t, yaml.Unmarshal([]byte(mergedYaml), &mergedConfig))
	assert.Contains(t, mergedConfig.Receivers, "kubeletstats")

	k8sAttr, ok := mergedConfig.Processors["k8sattributes/own-kubelet"].(map[string]interface{})
	assert.True(t, ok, "k8sattributes/own-kubelet should be configured")
	filter, ok := k8sAttr["filter"].(map[string]interface{})
	assert.True(t, ok, "k8sattributes/own-kubelet must scope its informers with a filter")
	assert.Equal(t, "odigos-system", filter["namespace"])
	assert.Equal(t, k8sconsts.NodeNameEnvVar, filter["node_from_env_var"])
}

func TestCalculateConfigMapDataTracesOnlyNoLoadBalancing(t *testing.T) {
	want := openTestData(t, "testdata/traces_only_no_loadbalancing.yaml")

	ns := NewMockNamespace("default")
	ns2 := NewMockNamespace("other-namespace")

	items := []v1alpha1.InstrumentationConfig{
		*NewMockInstrumentationConfig(NewMockTestDeployment(ns)),
		*NewMockInstrumentationConfig(NewMockTestDaemonSet(ns)),
		*NewMockInstrumentationConfig(NewMockTestStatefulSet(ns2)),
		*NewMockInstrumentationConfigWoOwner(NewMockTestDeployment(ns2)),
	}

	trueVal2 := true
	falseVal2 := false

	_, got, err := calculateCollectorConfigDomains(
		context.Background(),
		"odigos-system",
		&odigosv1.CollectorsGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "test-collector-group"},
			Spec: odigosv1.CollectorsGroupSpec{
				CollectorOwnMetricsPort: 4317,
				ResourceDetectors: &common.ResourceDetectorsConfiguration{
					EC2:   &common.ResourceDetectorConfig{Enabled: &trueVal2},
					EKS:   &common.ResourceDetectorConfig{Enabled: &falseVal2},
					Azure: &common.ResourceDetectorConfig{Enabled: &trueVal2},
					AKS:   &common.ResourceDetectorConfig{Enabled: &trueVal2},
				},
				// No metrics configuration - only traces
			},
		},
		&odigosv1.InstrumentationConfigList{
			Items: items,
		},
		[]common.ObservabilitySignal{
			// Only traces enabled, no logs or metrics
			common.TracesObservabilitySignal,
		},
		[]*v1alpha1.Processor{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "test_processor"},
				Spec: v1alpha1.ProcessorSpec{
					OrderHint:      1,
					Type:           "test_type",
					CollectorRoles: []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleNodeCollector},
					Disabled:       false,
					ProcessorConfig: runtime.RawExtension{
						Raw: []byte(`{"key":"val"}`),
					},
					Signals: []common.ObservabilitySignal{
						common.TracesObservabilitySignal,
					},
				},
			},
		},
		false,                   /* onGKE */
		false,                   /* loadBalancingNeeded */
		nil,                     /* profiling */
		common.OnPremOdigosTier, /* tier */
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, want, got)
}
