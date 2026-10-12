package nodecollector

import (
	"context"
	"os"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/instrumentor/controllers/collectorhooks"
	"github.com/odigos-io/odigos/instrumentor/controllers/nodecollector/collectorconfig"
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
	"github.com/stretchr/testify/require"
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
		false, /* onGKE */
		true,  /* loadBalancingNeeded */
		common.OdigosConfiguration{},
		nil, /* hooks */
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, want, got)
}

func boolPtr(b bool) *bool {
	return &b
}

func TestIsTracingLoadBalancingNeeded(t *testing.T) {
	activeTailSampling := &sampling.TailSamplingConfiguration{Disabled: boolPtr(false)}

	for _, tt := range []struct {
		name string
		spec odigosv1.CollectorsGroupSpec
		want bool
	}{
		{
			name: "defaults - service graph is enabled",
			spec: odigosv1.CollectorsGroupSpec{},
			want: true,
		},
		{
			name: "service graph disabled and nothing else aggregates traces",
			spec: odigosv1.CollectorsGroupSpec{ServiceGraphDisabled: boolPtr(true)},
			want: false,
		},
		{
			name: "service graph disabled but tail sampling is active",
			spec: odigosv1.CollectorsGroupSpec{
				ServiceGraphDisabled: boolPtr(true),
				TailSampling:         activeTailSampling,
			},
			want: true,
		},
		{
			name: "service graph disabled and tail sampling is disabled",
			spec: odigosv1.CollectorsGroupSpec{
				ServiceGraphDisabled: boolPtr(true),
				TailSampling:         &sampling.TailSamplingConfiguration{Disabled: boolPtr(true)},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isTracingLoadBalancingNeeded(context.Background(), nil, odigosv1.CollectorsGroup{Spec: tt.spec})
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Tail sampling groups spans by trace id on the gateway, so the node collectors must route every
// span of a trace to the same gateway replica even when the service graph is turned off.
func TestNodeCollectorUsesLoadBalancingExporterForTailSampling(t *testing.T) {
	clusterCollectorGroup := odigosv1.CollectorsGroup{
		Spec: odigosv1.CollectorsGroupSpec{
			ServiceGraphDisabled: boolPtr(true),
			TailSampling:         &sampling.TailSamplingConfiguration{Disabled: boolPtr(false)},
		},
	}

	loadBalancingNeeded, err := isTracingLoadBalancingNeeded(context.Background(), nil, clusterCollectorGroup)
	assert.NoError(t, err)

	_, got, err := calculateCollectorConfigDomains(
		context.Background(),
		"odigos-system",
		&odigosv1.CollectorsGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "test-collector-group"},
			Spec:       odigosv1.CollectorsGroupSpec{CollectorOwnMetricsPort: 4317},
		},
		&odigosv1.InstrumentationConfigList{
			Items: []v1alpha1.InstrumentationConfig{
				*NewMockInstrumentationConfig(NewMockTestDeployment(NewMockNamespace("default"))),
			},
		},
		[]common.ObservabilitySignal{common.TracesObservabilitySignal},
		nil,                 /* processors */
		false,               /* onGKE */
		loadBalancingNeeded, /* loadBalancingNeeded */
		common.OdigosConfiguration{},
		nil, /* hooks */
	)

	assert.NoError(t, err)

	collectorConfig := config.Config{}
	assert.NoError(t, yaml.Unmarshal([]byte(got), &collectorConfig))
	assert.Equal(t, []string{"loadbalancing/traces"}, collectorConfig.Service.Pipelines["traces"].Exporters)
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
		false, /* onGKE */
		false, /* loadBalancingNeeded */
		common.OdigosConfiguration{},
		nil, /* hooks */
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, want, got)
}

const hookReceiverName = "test/hook-receiver"

type nodeConfigHook struct{}

func (nodeConfigHook) MutateConfig(in collectorhooks.NodeCollectorHookInput) error {
	in.AddReceiver(hookReceiverName, config.GenericMap{})
	in.AppendReceiverToTraces(hookReceiverName)
	in.SetLogsReceivers([]string{hookReceiverName})
	return nil
}

func nodeCollectorConfig(t *testing.T, signals []common.ObservabilitySignal, hooks *collectorhooks.CollectorHooks) config.Config {
	t.Helper()
	_, got, err := calculateCollectorConfigDomains(
		context.Background(),
		"odigos-system",
		&odigosv1.CollectorsGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "test-collector-group"},
			Spec:       odigosv1.CollectorsGroupSpec{CollectorOwnMetricsPort: 4317},
		},
		&odigosv1.InstrumentationConfigList{
			Items: []v1alpha1.InstrumentationConfig{
				*NewMockInstrumentationConfig(NewMockTestDeployment(NewMockNamespace("default"))),
			},
		},
		signals,
		nil,   /* processors */
		false, /* onGKE */
		false, /* loadBalancingNeeded */
		common.OdigosConfiguration{},
		hooks,
	)
	require.NoError(t, err)
	var cfg config.Config
	require.NoError(t, yaml.Unmarshal([]byte(got), &cfg))
	return cfg
}

func anyPipelineHasAllReceivers(cfg config.Config, names ...string) bool {
	for _, p := range cfg.Service.Pipelines {
		ok := true
		for _, name := range names {
			if !containsString(p.Receivers, name) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func anyPipelineReceiversEqual(cfg config.Config, names []string) bool {
	for _, p := range cfg.Service.Pipelines {
		if len(p.Receivers) != len(names) {
			continue
		}
		match := true
		for i := range names {
			if p.Receivers[i] != names[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func TestNodeCollectorHook_NilIsCommunity(t *testing.T) {
	cfg := nodeCollectorConfig(t, []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.LogsObservabilitySignal,
	}, nil)
	assert.NotContains(t, cfg.Receivers, hookReceiverName)
	assert.False(t, anyPipelineHasAllReceivers(cfg, hookReceiverName))
}

func TestNodeCollectorHook_MutatesMergedConfig(t *testing.T) {
	hooks := &collectorhooks.CollectorHooks{NodeCollectorConfig: nodeConfigHook{}}
	cfg := nodeCollectorConfig(t, []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.LogsObservabilitySignal,
	}, hooks)

	assert.Contains(t, cfg.Receivers, hookReceiverName)
	assert.True(t, anyPipelineHasAllReceivers(cfg, collectorconfig.OTLPInReceiverName, hookReceiverName),
		"hook receiver should be appended alongside the community traces receiver")
	assert.True(t, anyPipelineReceiversEqual(cfg, []string{hookReceiverName}),
		"hook should replace the community logs receivers")
}

func TestNodeCollectorHook_MissingLogsPipelineIsNoop(t *testing.T) {
	hooks := &collectorhooks.CollectorHooks{NodeCollectorConfig: nodeConfigHook{}}
	cfg := nodeCollectorConfig(t, []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
	}, hooks)

	assert.Contains(t, cfg.Receivers, hookReceiverName)
	assert.True(t, anyPipelineHasAllReceivers(cfg, collectorconfig.OTLPInReceiverName, hookReceiverName))
	assert.False(t, anyPipelineReceiversEqual(cfg, []string{hookReceiverName}),
		"SetLogsReceivers must not create a logs pipeline")
}
