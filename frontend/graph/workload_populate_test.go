package graph

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/odigos-io/odigos/frontend/kube"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	frontendcommon "github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	generatedstatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	wpOdigosNamespace = "odigos-system"
	wpAppNamespace    = "app-ns"
	wpWorkloadName    = "inventory"
	// The ReplicaSet name a Deployment's pods are owned by. workload.PodWorkloadObject
	// resolves ReplicaSet -> Deployment by stripping everything after the last hyphen.
	wpReplicaSetName = wpWorkloadName + "-7d4c8b5f9b"
	// A distro that reports InstrumentationInstances, and one that ships but does not.
	wpReportingDistro = "golang-community"
	wpSilentDistro    = "dotnet-community"
)

// wpRuntimeObjectName is the name odigos derives for the InstrumentationConfig and
// carries on the InstrumentedAppName label of every InstrumentationInstance.
const wpRuntimeObjectName = "deployment-" + wpWorkloadName

func wpBool(b bool) *bool { return &b }

// wpContainer describes a container in the terms populateWorkloadFields branches on
// rather than in Kubernetes terms; wpBuild turns each field into the manifest that
// produces it, so the real loading path stays inside the contract.
type wpContainer struct {
	name string
	// distro becomes the ODIGOS_DISTRO_NAME env var. Empty means no agent was injected.
	distro string
	// ready becomes the pod's ContainerStatus.Ready.
	ready bool
	// instanceHealthy is the InstrumentationInstance health. Absent means no instance
	// was reported for this container at all.
	instanceHealthy   *bool
	hasInstance       bool
	instanceMessage   string
	instanceComponent *odigosv1.InstrumentationLibraryStatus
}

type wpFixture struct {
	availableReplicas int32
	// ic is the workload's InstrumentationConfig. Absent means the workload is not
	// instrumented, which is the branch most of populateWorkloadFields is guarded on.
	ic *odigosv1.InstrumentationConfig
	// source is the workload's Source CR. Absent means no Source at all.
	source     *odigosv1.Source
	containers []wpContainer
	tier       model.Tier
	// agentsMetaHash, when set on both the InstrumentationConfig and the pod label,
	// is what makes odigos consider the pod successfully injected.
	agentsMetaHash string
	// reportingTelemetry drives a real metrics consumer that has actually ingested
	// traffic for this workload, which is what lets it reach the "telemetry is being
	// collected" steady state.
	reportingTelemetry bool
}

func wpScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, odigosv1.AddToScheme(s))
	return s
}

// wpIgnoredContainer is a container name the installation is configured to ignore. It
// is a real odigos default, and languages detected in it must never reach the UI.
const wpIgnoredContainer = "istio-proxy"

// wpEffectiveConfigMap must carry non-empty YAML: an empty body unmarshals the
// *common.OdigosConfiguration pointer to nil and loadWorkloadPods then nil-derefs on it.
func wpEffectiveConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: wpOdigosNamespace, Name: consts.OdigosEffectiveConfigName},
		Data: map[string]string{consts.OdigosConfigurationFileName: "configVersion: 1\nignoredContainers:\n  - " +
			wpIgnoredContainer + "\n"},
	}
}

func wpDeployment(availableReplicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: wpAppNamespace, Name: wpWorkloadName},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": wpWorkloadName}},
		},
		Status: appsv1.DeploymentStatus{AvailableReplicas: availableReplicas},
	}
}

func wpPod(containers []wpContainer, agentsMetaHash string) *corev1.Pod {
	labels := map[string]string{"app": wpWorkloadName}
	if agentsMetaHash != "" {
		labels[k8sconsts.OdigosAgentsMetaHashLabel] = agentsMetaHash
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wpAppNamespace,
			Name:      wpReplicaSetName + "-abcde",
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: wpReplicaSetName, APIVersion: "apps/v1"},
			},
		},
	}
	for _, c := range containers {
		container := corev1.Container{Name: c.name}
		if c.distro != "" {
			container.Env = []corev1.EnvVar{{Name: k8sconsts.OdigosEnvVarDistroName, Value: c.distro}}
		}
		pod.Spec.Containers = append(pod.Spec.Containers, container)
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:  c.name,
			Ready: c.ready,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.Now()}},
		})
	}
	return pod
}

func wpInstanceObjects(podName string, containers []wpContainer) []client.Object {
	objects := []client.Object{}
	for _, c := range containers {
		if !c.hasInstance {
			continue
		}
		instance := &odigosv1.InstrumentationInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: wpAppNamespace,
				Name:      podName + "-" + c.name,
				Labels: map[string]string{
					odigosv1.OwnerPodNameLabel:      podName,
					consts.InstrumentedAppNameLabel: wpRuntimeObjectName,
				},
			},
			Spec: odigosv1.InstrumentationInstanceSpec{ContainerName: c.name},
			Status: odigosv1.InstrumentationInstanceStatus{
				Healthy: c.instanceHealthy,
				Message: c.instanceMessage,
			},
		}
		if c.instanceComponent != nil {
			instance.Status.Components = []odigosv1.InstrumentationLibraryStatus{*c.instanceComponent}
		}
		objects = append(objects, instance)
	}
	return objects
}

func wpWorkloadID() model.K8sWorkloadID {
	return model.K8sWorkloadID{
		Namespace: wpAppNamespace,
		Kind:      model.K8sResourceKindDeployment,
		Name:      wpWorkloadName,
	}
}

// The frontend keeps its Kubernetes client and its metrics consumer in package globals
// that production writes exactly once at startup. The metrics consumer's delete watcher
// reads kube.DefaultClient from a client-go goroutine that is never joined, so writing
// that global per test is a data race; both are therefore built once per test binary.
var (
	wpSharedOnce             sync.Once
	wpSharedMetricsConsumer  *collectormetrics.OdigosMetricsConsumer
	wpSharedMetricsSetupFail string
)

const (
	// wpDataSentBytes is the trace volume the shared consumer reports for the fixture
	// workload, spread over wpDataSentInterval.
	wpDataSentBytes = 4096
	// The interval is long enough that the integer throughput rounds down to zero: a
	// workload that has sent data but is idle right now. Telemetry must still count as
	// observed, which is what distinguishes the total from the current rate.
	wpDataSentInterval = 2 * time.Hour
)

func wpEnsureSharedState(t *testing.T) {
	t.Helper()
	wpSharedOnce.Do(func() {
		kube.SetDefaultClient(&kube.Client{Interface: k8sfake.NewSimpleClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: wpOdigosNamespace, Name: k8sconsts.OdigosDeploymentConfigMapName},
			Data:       map[string]string{k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.CommunityOdigosTier)},
		})})
		wpSharedMetricsConsumer, wpSharedMetricsSetupFail = wpBuildReportingConsumer()
	})
	require.Empty(t, wpSharedMetricsSetupFail)
}

// wpBuildReportingConsumer drives the production ingestion path: the source is
// registered through the notification loop and two OTLP batches are consumed, since the
// first only establishes the baseline the second is measured against.
func wpBuildReportingConsumer() (*collectormetrics.OdigosMetricsConsumer, string) {
	consumer := collectormetrics.NewOdigosMetrics()
	// The loop runs for the lifetime of the test binary. Cancelling it per test would
	// mean restoring kube.DefaultClient underneath the watcher goroutine.
	go consumer.RunDeleteWatcherAndNotifications(context.Background(), wpOdigosNamespace)

	sourceID := frontendcommon.SourceID{
		Namespace: wpAppNamespace,
		Kind:      k8sconsts.WorkloadKindDeployment,
		Name:      wpWorkloadName,
	}
	consumer.NotifySourceAdded(sourceID)
	registered := false
	for range 10000 {
		if _, tracked := consumer.GetSingleSourceMetrics(sourceID); tracked {
			registered = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !registered {
		return nil, "the notification loop never registered the source"
	}

	baseTime := time.Unix(1_800_000_000, 0).UTC()
	for i, value := range []float64{0, wpDataSentBytes} {
		if err := consumer.ConsumeMetrics(context.Background(),
			wpTraceSizeMetric(value, baseTime.Add(time.Duration(i)*wpDataSentInterval))); err != nil {
			return nil, "consuming the trace size metric failed: " + err.Error()
		}
	}
	metrics, ok := consumer.GetSingleSourceMetrics(sourceID)
	if !ok || metrics.TotalDataSent() != wpDataSentBytes {
		return nil, "the consumer did not record the reported trace volume"
	}
	if metrics.TotalThroughput() != 0 {
		return nil, "the fixture is meant to report zero current throughput"
	}
	return consumer, ""
}

// wpLoaderContext seeds a fake cluster from the fixture and drives the real
// LoadWorkloadsWithFilter priming step, so everything downstream exercises the
// production loading path rather than a hand-stuffed cache.
func wpLoaderContext(t *testing.T, f wpFixture) context.Context {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, wpOdigosNamespace)
	wpEnsureSharedState(t)

	pod := wpPod(f.containers, f.agentsMetaHash)
	objects := []client.Object{wpEffectiveConfigMap(), wpDeployment(f.availableReplicas), pod}
	if f.ic != nil {
		objects = append(objects, f.ic)
	}
	if f.source != nil {
		objects = append(objects, f.source)
	}
	objects = append(objects, wpInstanceObjects(pod.Name, f.containers)...)

	c := fake.NewClientBuilder().WithScheme(wpScheme(t)).WithObjects(objects...).Build()
	l := loaders.NewLoaders(logr.Discard(), c)

	ctx := context.Background()
	namespace, name := wpAppNamespace, wpWorkloadName
	kind := model.K8sResourceKindDeployment
	require.NoError(t, l.LoadWorkloadsWithFilter(ctx, &model.WorkloadFilter{
		Namespace: &namespace, Kind: &kind, Name: &name,
	}))
	return loaders.WithLoaders(ctx, l)
}

// wpMetricsConsumer returns the shared consumer that has ingested telemetry for the
// fixture workload, or a zero-value consumer that tracks no sources at all — in which
// case GetSingleSourceMetrics reports "not found" and totalDataSent stays nil, the state
// every workload is in until its collectors report.
func wpMetricsConsumer(t *testing.T, reportingTelemetry bool) *collectormetrics.OdigosMetricsConsumer {
	t.Helper()
	if !reportingTelemetry {
		return &collectormetrics.OdigosMetricsConsumer{}
	}
	wpEnsureSharedState(t)
	return wpSharedMetricsConsumer
}

// wpTraceSizeMetric builds the OTLP batch a node collector sends for one workload's
// trace volume, using the exact resource attributes ConsumeMetrics routes on.
func wpTraceSizeMetric(value float64, at time.Time) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("odigos.collector.role", string(k8sconsts.CollectorsRoleNodeCollector))
	rm.Resource().Attributes().PutStr("k8s.pod.name", "odigos-data-collection-abcde")
	metric := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("otelcol_odigos_trace_data_size_bytes_total")
	dp := metric.SetEmptySum().DataPoints().AppendEmpty()
	dp.SetDoubleValue(value)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(at))
	dp.Attributes().PutStr("k8s.namespace.name", wpAppNamespace)
	dp.Attributes().PutStr("k8s.deployment.name", wpWorkloadName)
	return md
}

// wpHarness returns a resolver and a request context wired to the same fake cluster,
// so the eager and the lazy code paths can be driven against identical inputs.
func wpHarness(t *testing.T, f wpFixture) (*Resolver, context.Context) {
	t.Helper()
	ctx := wpLoaderContext(t, f)
	return &Resolver{MetricsConsumer: wpMetricsConsumer(t, f.reportingTelemetry)}, ctx
}

func wpPopulate(t *testing.T, f wpFixture) *model.K8sWorkload {
	t.Helper()
	r, ctx := wpHarness(t, f)
	id := wpWorkloadID()
	w := &model.K8sWorkload{ID: &id}
	(&queryResolver{r}).populateWorkloadFields(ctx, loaders.For(ctx), w, f.tier)
	return w
}

// wpIC builds an InstrumentationConfig whose Spec.Containers mirror the fixture's
// containers, which is what the containers / agentEnabled fields are derived from.
func wpIC(serviceName string, containers []wpContainer) *odigosv1.InstrumentationConfig {
	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Namespace: wpAppNamespace, Name: wpRuntimeObjectName},
		Spec: odigosv1.InstrumentationConfigSpec{
			ServiceName:           serviceName,
			AgentInjectionEnabled: true,
		},
	}
	for _, c := range containers {
		ic.Spec.Containers = append(ic.Spec.Containers, odigosv1.ContainerAgentConfig{
			ContainerName:  c.name,
			AgentEnabled:   c.distro != "",
			OtelDistroName: c.distro,
		})
		ic.Spec.ContainersOverrides = append(ic.Spec.ContainersOverrides, odigosv1.ContainerOverride{
			ContainerName: c.name,
		})
	}
	return ic
}

func wpWorkloadSource(otelServiceName string, disabled bool) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wpAppNamespace,
			Name:      "source-" + wpWorkloadName,
			Labels: map[string]string{
				k8sconsts.WorkloadNamespaceLabel: wpAppNamespace,
				k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindDeployment),
				k8sconsts.WorkloadNameLabel:      wpWorkloadName,
			},
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: wpAppNamespace,
				Kind:      k8sconsts.WorkloadKindDeployment,
				Name:      wpWorkloadName,
			},
			OtelServiceName:        otelServiceName,
			DisableInstrumentation: disabled,
		},
	}
}

// wpHealthyContainer is the happy path: an agent that reports instrumentation instances,
// a ready container, and a healthy instance with one healthy library.
func wpHealthyContainer(name string) wpContainer {
	return wpContainer{
		name:            name,
		distro:          wpReportingDistro,
		ready:           true,
		hasInstance:     true,
		instanceHealthy: wpBool(true),
		instanceComponent: &odigosv1.InstrumentationLibraryStatus{
			Name:    "net/http",
			Type:    odigosv1.InstrumentationLibraryTypeInstrumentation,
			Healthy: wpBool(true),
		},
	}
}

func wpInstrumentedFixture() wpFixture {
	containers := []wpContainer{wpHealthyContainer("app")}
	return wpFixture{
		availableReplicas: 3,
		ic:                wpIC("inventory-service", containers),
		source:            wpWorkloadSource("", false),
		containers:        containers,
	}
}

// wpReconciledConditions are the four InstrumentationConfig status conditions the
// instrumentor writes once it has finished reconciling a workload successfully.
func wpReconciledConditions() []metav1.Condition {
	condition := func(conditionType, reason string) metav1.Condition {
		return metav1.Condition{
			Type:               conditionType,
			Status:             metav1.ConditionTrue,
			Reason:             reason,
			LastTransitionTime: metav1.Now(),
		}
	}
	return []metav1.Condition{
		condition(odigosv1.RuntimeDetectionStatusConditionType, string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully)),
		condition(generatedstatus.AgentEnabledType, string(odigosv1.AgentEnabledReasonEnabledSuccessfully)),
		condition(generatedstatus.PodsManifestInjectionType, string(generatedstatus.PodsManifestInjectionReasonPodsAppliedSuccessfully_Enabled)),
		condition(odigosv1.WorkloadRolloutStatusConditionType, string(odigosv1.WorkloadRolloutReasonRolloutFinished)),
	}
}

// wpSteadyStateFixture is a workload in the state every working installation settles
// into: reconciled, rolled out, agent injected with a matching hash, healthy processes
// and telemetry actually arriving. This is the only shape that reaches the aggregated
// "collecting telemetry" health, so it is the one that proves the happy path exists.
func wpSteadyStateFixture() wpFixture {
	containers := []wpContainer{wpHealthyContainer("app")}
	ic := wpIC("inventory-service", containers)
	ic.Spec.AgentsMetaHash = "hash-v1"
	ic.Status.Conditions = wpReconciledConditions()
	ic.Status.RuntimeDetailsByContainer = []odigosv1.RuntimeDetailsByContainer{
		{ContainerName: "app", Language: "go"},
	}
	return wpFixture{
		availableReplicas:  3,
		ic:                 ic,
		source:             wpWorkloadSource("", false),
		containers:         containers,
		agentsMetaHash:     "hash-v1",
		reportingTelemetry: true,
	}
}

const wpStaticPodName = "etcd-node-1"

// wpPopulateStaticPod seeds a cluster holding a real static pod — owned by a Node and
// carrying the kubelet's config-source annotation, which is what odigos keys on — and
// runs the eager pass over it.
func wpPopulateStaticPod(t *testing.T, tier model.Tier) *model.K8sWorkload {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, wpOdigosNamespace)
	wpEnsureSharedState(t)

	staticPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       wpAppNamespace,
			Name:            wpStaticPodName,
			Labels:          map[string]string{k8sconsts.OdigosVirtualStaticPodNameLabel: wpStaticPodName},
			Annotations:     map[string]string{"kubernetes.io/config.source": "file"},
			OwnerReferences: []metav1.OwnerReference{{Kind: "Node", Name: "node-1", APIVersion: "v1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "etcd",
			Env:  []corev1.EnvVar{{Name: k8sconsts.OdigosEnvVarDistroName, Value: wpReportingDistro}},
		}}},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: "etcd", Ready: true}}},
	}
	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wpAppNamespace,
			Name:      workload.CalculateWorkloadRuntimeObjectName(wpStaticPodName, k8sconsts.WorkloadKindStaticPod),
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ServiceName:           "etcd",
			AgentInjectionEnabled: true,
			Containers:            []odigosv1.ContainerAgentConfig{{ContainerName: "etcd", AgentEnabled: true}},
		},
	}

	c := fake.NewClientBuilder().WithScheme(wpScheme(t)).
		WithObjects(wpEffectiveConfigMap(), staticPod, ic).Build()
	l := loaders.NewLoaders(logr.Discard(), c)

	ctx := context.Background()
	namespace, name := wpAppNamespace, wpStaticPodName
	kind := model.K8sResourceKindStaticPod
	require.NoError(t, l.LoadWorkloadsWithFilter(ctx, &model.WorkloadFilter{
		Namespace: &namespace, Kind: &kind, Name: &name,
	}))
	ctx = loaders.WithLoaders(ctx, l)

	id := model.K8sWorkloadID{Namespace: wpAppNamespace, Kind: model.K8sResourceKindStaticPod, Name: wpStaticPodName}
	w := &model.K8sWorkload{ID: &id}
	r := &queryResolver{&Resolver{MetricsConsumer: &collectormetrics.OdigosMetricsConsumer{}}}
	r.populateWorkloadFields(ctx, loaders.For(ctx), w, tier)
	return w
}

func TestPopulateWorkloadFields_AnInstrumentedWorkloadGetsEveryFieldPrecomputed(t *testing.T) {
	w := wpPopulate(t, wpInstrumentedFixture())

	require.NotNil(t, w.ServiceName)
	assert.Equal(t, "inventory-service", *w.ServiceName)
	require.NotNil(t, w.NumberOfInstances)
	assert.Equal(t, 3, *w.NumberOfInstances)
	require.NotNil(t, w.MarkedForInstrumentation)
	require.NotNil(t, w.MarkedForInstrumentation.MarkedForInstrumentation)
	assert.True(t, *w.MarkedForInstrumentation.MarkedForInstrumentation)
	assert.NotNil(t, w.DataStreamNames)
	require.NotNil(t, w.RuntimeInfo)
	require.Len(t, w.Containers, 1)
	assert.Equal(t, "app", w.Containers[0].ContainerName)
	require.NotNil(t, w.Conditions)
	assert.NotNil(t, w.PodsAgentInjectionStatus)
	require.NotNil(t, w.WorkloadOdigosHealthStatus)
}

// Every pre-computed field exists so the matching lazy resolver can short-circuit on
// `if obj.X != nil`. A field populate forgets to set silently falls back to the lazy
// path, which re-reads the cluster once per workload — the exact blow-up the eager
// pass was written to avoid. This pins the full set.
func TestPopulateWorkloadFields_EveryShortCircuitableFieldIsActuallyPopulated(t *testing.T) {
	w := wpPopulate(t, wpInstrumentedFixture())

	for _, tt := range []struct {
		field string
		set   bool
	}{
		{"serviceName", w.ServiceName != nil},
		{"markedForInstrumentation", w.MarkedForInstrumentation != nil},
		{"dataStreamNames", w.DataStreamNames != nil},
		{"numberOfInstances", w.NumberOfInstances != nil},
		{"runtimeInfo", w.RuntimeInfo != nil},
		{"containers", w.Containers != nil},
		{"conditions", w.Conditions != nil},
		{"podsAgentInjectionStatus", w.PodsAgentInjectionStatus != nil},
		{"workloadOdigosHealthStatus", w.WorkloadOdigosHealthStatus != nil},
	} {
		assert.True(t, tt.set, "%s must be pre-computed so its resolver short-circuits", tt.field)
	}
}

// Disabling a Source deletes the InstrumentationConfig, so the configured otel service
// name is only reachable from the Source CR. Losing this fallback makes the service
// name disappear from the UI the moment a user disables a source.
func TestPopulateWorkloadFields_TheServiceNameFallsBackToTheSourceWhenThereIsNoInstrumentationConfig(t *testing.T) {
	f := wpInstrumentedFixture()
	f.ic = nil
	f.source = wpWorkloadSource("service-from-source", true)

	w := wpPopulate(t, f)

	require.NotNil(t, w.ServiceName)
	assert.Equal(t, "service-from-source", *w.ServiceName)
}

// The InstrumentationConfig is authoritative while it exists; the Source fallback must
// not overwrite it.
func TestPopulateWorkloadFields_TheInstrumentationConfigServiceNameWinsOverTheSource(t *testing.T) {
	f := wpInstrumentedFixture()
	f.source = wpWorkloadSource("service-from-source", false)

	w := wpPopulate(t, f)

	require.NotNil(t, w.ServiceName)
	assert.Equal(t, "inventory-service", *w.ServiceName)
}

// markedForInstrumentation is a tri-state: true (instrumented), false (explicitly
// disabled by the user) and nil (never marked). The UI renders a different control for
// each, so collapsing "disabled" into "never marked" is a user-visible regression.
func TestPopulateWorkloadFields_MarkedForInstrumentationIsTriState(t *testing.T) {
	for _, tt := range []struct {
		name       string
		source     *odigosv1.Source
		withIC     bool
		wantMarked *bool
	}{
		{"an enabled source marks the workload", wpWorkloadSource("", false), true, wpBool(true)},
		{"an explicitly disabled source reports false, not unknown", wpWorkloadSource("", true), false, wpBool(false)},
		{"no source at all leaves the decision unknown", nil, false, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := wpInstrumentedFixture()
			f.source = tt.source
			if !tt.withIC {
				f.ic = nil
			}

			w := wpPopulate(t, f)

			require.NotNil(t, w.MarkedForInstrumentation)
			assert.Equal(t, tt.wantMarked, w.MarkedForInstrumentation.MarkedForInstrumentation)
			assert.NotEmpty(t, w.MarkedForInstrumentation.DecisionEnum, "the UI always renders the decision reason")
		})
	}
}

// dataStreamNames backs a non-null GraphQL list. Returning nil rather than an empty
// slice makes gqlgen fail the whole workload.
func TestPopulateWorkloadFields_DataStreamNamesIsAnEmptyNonNilSliceWhenTheSourceHasNoStreams(t *testing.T) {
	w := wpPopulate(t, wpInstrumentedFixture())

	require.NotNil(t, w.DataStreamNames)
	assert.Empty(t, w.DataStreamNames)
}

// Data stream membership is encoded as labels on the Source CR and is what the UI
// groups workloads by. The names are copied out of a []*string, so a wrong index in
// that copy silently assigns one workload to another workload's stream.
func TestPopulateWorkloadFields_DataStreamNamesAreReadFromTheSourceLabels(t *testing.T) {
	f := wpInstrumentedFixture()
	source := wpWorkloadSource("", false)
	source.Labels[k8sconsts.SourceDataStreamLabelPrefix+"payments"] = "true"
	source.Labels[k8sconsts.SourceDataStreamLabelPrefix+"checkout"] = "true"
	source.Labels[k8sconsts.SourceDataStreamLabelPrefix+"legacy"] = "false"
	f.source = source

	w := wpPopulate(t, f)

	assert.ElementsMatch(t, []string{"payments", "checkout"}, w.DataStreamNames)
	assert.NotContains(t, w.DataStreamNames, "legacy", "a label set to false excludes the stream")
}

// Without an InstrumentationConfig the whole instrumented half of the workload is
// unknown, and the fields the UI keys off must stay nil rather than render as empty
// (which reads as "instrumented with nothing").
func TestPopulateWorkloadFields_AnUninstrumentedWorkloadLeavesTheInstrumentedFieldsUnset(t *testing.T) {
	f := wpInstrumentedFixture()
	f.ic = nil
	f.source = nil

	w := wpPopulate(t, f)

	assert.Nil(t, w.RuntimeInfo)
	assert.Nil(t, w.Containers)
	assert.Nil(t, w.Conditions)
	assert.Nil(t, w.ServiceName)
	assert.False(t, w.RollbackOccurred)
	require.NotNil(t, w.NumberOfInstances, "the replica count comes from the manifest and is always known")
	require.NotNil(t, w.WorkloadOdigosHealthStatus, "an uninstrumented workload still needs a health status")
}

// An uninstrumented workload reports "disabled" health, and the message must come from
// the source decision when there is one — that message is what tells the user *why*.
func TestPopulateWorkloadFields_AnUninstrumentedWorkloadReportsDisabledHealth(t *testing.T) {
	f := wpInstrumentedFixture()
	f.ic = nil
	f.source = wpWorkloadSource("", true)

	w := wpPopulate(t, f)

	require.NotNil(t, w.WorkloadOdigosHealthStatus)
	assert.Equal(t, model.DesiredStateProgressDisabled, w.WorkloadOdigosHealthStatus.Status)
	require.NotNil(t, w.WorkloadOdigosHealthStatus.ReasonEnum)
	assert.Equal(t, string(status.WorkloadOdigosHealthStatusReasonDisabled), *w.WorkloadOdigosHealthStatus.ReasonEnum)
	require.NotNil(t, w.MarkedForInstrumentation)
	assert.Equal(t, w.MarkedForInstrumentation.Message, w.WorkloadOdigosHealthStatus.Message,
		"the health message must explain the source decision rather than repeat a generic string")
}

// Even with no Source CR at all, IsObjectInstrumentedBySource still produces a decision
// message, so the disabled-health message is always the source explanation and the
// generic "workload is not marked for instrumentation" literal is unreachable here.
// The lazy WorkloadOdigosHealthStatus resolver hardcodes that literal instead — the
// divergence is pinned in workload_populate_parity_test.go.
func TestPopulateWorkloadFields_TheDisabledHealthMessageAlwaysExplainsTheSourceDecision(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source *odigosv1.Source
	}{
		{"with a disabled source", wpWorkloadSource("", true)},
		{"with no source at all", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := wpInstrumentedFixture()
			f.ic = nil
			f.source = tt.source

			w := wpPopulate(t, f)

			require.NotNil(t, w.MarkedForInstrumentation)
			require.NotEmpty(t, w.MarkedForInstrumentation.Message)
			require.NotNil(t, w.WorkloadOdigosHealthStatus)
			assert.Equal(t, w.MarkedForInstrumentation.Message, w.WorkloadOdigosHealthStatus.Message)
			assert.NotEqual(t, "workload is not marked for instrumentation", w.WorkloadOdigosHealthStatus.Message,
				"the generic default must not shadow the specific source explanation")
		})
	}
}

// A fully reconciled workload that is reporting telemetry must reach the dedicated
// "collecting telemetry" state — a single aggregated green badge — rather than surface
// whichever individual success condition happened to win the severity aggregation.
func TestPopulateWorkloadFields_AWorkloadInSteadyStateReportsCollectingTelemetry(t *testing.T) {
	w := wpPopulate(t, wpSteadyStateFixture())

	require.NotNil(t, w.WorkloadOdigosHealthStatus)
	assert.Equal(t, model.DesiredStateProgressSuccess, w.WorkloadOdigosHealthStatus.Status)
	require.NotNil(t, w.WorkloadOdigosHealthStatus.ReasonEnum)
	assert.Equal(t, string(status.WorkloadOdigosHealthStatusReasonCollectingTelemetry),
		*w.WorkloadOdigosHealthStatus.ReasonEnum)
	assert.Equal(t, "source is instrumented and telemetry is being collected", w.WorkloadOdigosHealthStatus.Message)
	assert.Equal(t, status.WorkloadOdigosHealthStatus, w.WorkloadOdigosHealthStatus.Name,
		"the aggregated status is renamed to the workload-level condition, not left as the winning sub-condition")
}

// Telemetry only counts as observed once bytes have actually been sent. Without the
// metrics feed the same fully reconciled workload must stay in "waiting", so the
// success state cannot be reached by a workload that is silently sending nothing.
func TestPopulateWorkloadFields_WithoutReportedTelemetryTheSteadyStateWorkloadStaysWaiting(t *testing.T) {
	f := wpSteadyStateFixture()
	f.reportingTelemetry = false

	w := wpPopulate(t, f)

	require.NotNil(t, w.Conditions)
	require.NotNil(t, w.Conditions.ExpectingTelemetry)
	assert.Equal(t, model.DesiredStateProgressWaiting, w.Conditions.ExpectingTelemetry.Status)
	require.NotNil(t, w.WorkloadOdigosHealthStatus)
	assert.NotEqual(t, model.DesiredStateProgressSuccess, w.WorkloadOdigosHealthStatus.Status)
}

// wpWithConditionReason rewrites one InstrumentationConfig status condition so the
// matching workload condition becomes the workload's worst problem.
func wpWithConditionReason(f wpFixture, conditionType, reason string) wpFixture {
	for i := range f.ic.Status.Conditions {
		if f.ic.Status.Conditions[i].Type == conditionType {
			f.ic.Status.Conditions[i].Reason = reason
			f.ic.Status.Conditions[i].Message = "something went wrong with " + conditionType
			return f
		}
	}
	panic("no such condition in the fixture: " + conditionType)
}

// The aggregated health is the single badge the workloads list shows per workload, and
// it has to be the worst of the seven conditions. Driving each condition to be the
// uniquely worst one and requiring the aggregate to be exactly it is what catches a
// condition being dropped from the aggregation — a workload rendering green while one
// specific thing about it is broken.
func TestPopulateWorkloadFields_EachConditionCanBecomeTheAggregatedHealth(t *testing.T) {
	for _, tt := range []struct {
		name     string
		fixture  func() wpFixture
		pick     func(*model.K8sWorkloadConditions) *model.DesiredConditionStatus
		severity model.DesiredStateProgress
	}{
		{
			name: "a runtime detection error",
			fixture: func() wpFixture {
				return wpWithConditionReason(wpSteadyStateFixture(),
					odigosv1.RuntimeDetectionStatusConditionType, string(odigosv1.RuntimeDetectionReasonError))
			},
			pick:     func(c *model.K8sWorkloadConditions) *model.DesiredConditionStatus { return c.RuntimeDetection },
			severity: model.DesiredStateProgressFailure,
		},
		{
			name: "another agent detected alongside odigos",
			fixture: func() wpFixture {
				return wpWithConditionReason(wpSteadyStateFixture(),
					generatedstatus.AgentEnabledType, string(odigosv1.AgentEnabledReasonOtherAgentDetected))
			},
			pick:     func(c *model.K8sWorkloadConditions) *model.DesiredConditionStatus { return c.AgentInjectionEnabled },
			severity: model.DesiredStateProgressNotice,
		},
		{
			name: "a rollout odigos could not patch",
			fixture: func() wpFixture {
				return wpWithConditionReason(wpSteadyStateFixture(),
					odigosv1.WorkloadRolloutStatusConditionType, string(odigosv1.WorkloadRolloutReasonFailedToPatch))
			},
			pick:     func(c *model.K8sWorkloadConditions) *model.DesiredConditionStatus { return c.Rollout },
			severity: model.DesiredStateProgressFailure,
		},
		{
			name: "a failed automatic rollout leaving pods without the agent",
			fixture: func() wpFixture {
				return wpWithConditionReason(wpSteadyStateFixture(), generatedstatus.PodsManifestInjectionType,
					string(generatedstatus.PodsManifestInjectionReasonRestartRequiredAutoRolloutFailed_Enabled))
			},
			pick:     func(c *model.K8sWorkloadConditions) *model.DesiredConditionStatus { return c.PodsManifestInjection },
			severity: model.DesiredStateProgressFailure,
		},
		{
			name: "an agent reporting itself unhealthy",
			fixture: func() wpFixture {
				f := wpSteadyStateFixture()
				containers := []wpContainer{{
					name: "app", distro: wpReportingDistro, ready: true,
					hasInstance: true, instanceHealthy: wpBool(false), instanceMessage: "agent crashed",
				}}
				f.containers = containers
				return f
			},
			pick:     func(c *model.K8sWorkloadConditions) *model.DesiredConditionStatus { return c.ProcessesAgentHealth },
			severity: model.DesiredStateProgressFailure,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := wpPopulate(t, tt.fixture())

			require.NotNil(t, w.Conditions)
			worst := tt.pick(w.Conditions)
			require.NotNil(t, worst)
			require.Equal(t, tt.severity, worst.Status,
				"precondition: this condition must be the uniquely most severe one")
			require.NotNil(t, w.WorkloadOdigosHealthStatus)
			assert.Equal(t, worst.Status, w.WorkloadOdigosHealthStatus.Status)
			assert.Equal(t, worst.Message, w.WorkloadOdigosHealthStatus.Message)
			assert.Equal(t, worst.ReasonEnum, w.WorkloadOdigosHealthStatus.ReasonEnum)
		})
	}
}

// A pod carrying a different agent hash than the InstrumentationConfig is running an
// out-of-date agent and must not be reported as successfully injected, or the UI tells
// users a rollout landed when it did not.
func TestPopulateWorkloadFields_AStaleAgentHashIsNotReportedAsInjected(t *testing.T) {
	f := wpSteadyStateFixture()
	f.agentsMetaHash = "hash-v0"

	w := wpPopulate(t, f)

	require.NotNil(t, w.Conditions)
	require.NotNil(t, w.Conditions.AgentInjected)
	assert.NotEqual(t, model.DesiredStateProgressSuccess, w.Conditions.AgentInjected.Status)
	require.NotNil(t, w.WorkloadOdigosHealthStatus)
	assert.NotEqual(t, model.DesiredStateProgressSuccess, w.WorkloadOdigosHealthStatus.Status)
}

// A static pod is the one workload kind whose health is a licensing decision. On a
// community installation populateWorkloadFields must stop at the gate and report
// "unsupported"; on on-prem the same workload must fall through to the real aggregation.
// Without the second half, removing the gate's early return would go unnoticed.
func TestPopulateWorkloadFields_TheStaticPodEnterpriseGateShortCircuitsOnCommunityOnly(t *testing.T) {
	for _, tt := range []struct {
		tier   model.Tier
		gated  bool
		reason status.WorkloadOdigosHealthStatusReason
	}{
		{tier: model.Tier("community"), gated: true, reason: status.WorkloadOdigosHealthStatusReasonEnterpriseFeature},
		{tier: model.TierOnprem},
	} {
		t.Run(string(tt.tier), func(t *testing.T) {
			w := wpPopulateStaticPod(t, tt.tier)

			require.NotNil(t, w.WorkloadOdigosHealthStatus)
			if !tt.gated {
				assert.NotEqual(t, model.DesiredStateProgressUnsupported, w.WorkloadOdigosHealthStatus.Status,
					"an on-prem static pod must be evaluated like any other workload")
				return
			}
			assert.Equal(t, model.DesiredStateProgressUnsupported, w.WorkloadOdigosHealthStatus.Status)
			require.NotNil(t, w.WorkloadOdigosHealthStatus.ReasonEnum)
			assert.Equal(t, string(tt.reason), *w.WorkloadOdigosHealthStatus.ReasonEnum)
		})
	}
}

// The gate is keyed on the workload kind, so a deployment must never be gated no matter
// what tier the installation is on.
func TestPopulateWorkloadFields_ADeploymentIsNeverGatedByTheEnterpriseCheck(t *testing.T) {
	for _, tier := range []model.Tier{model.Tier("community"), model.TierOnprem} {
		t.Run(string(tier), func(t *testing.T) {
			f := wpInstrumentedFixture()
			f.tier = tier

			w := wpPopulate(t, f)

			require.NotNil(t, w.WorkloadOdigosHealthStatus)
			assert.NotEqual(t, model.DesiredStateProgressUnsupported, w.WorkloadOdigosHealthStatus.Status)
		})
	}
}

// The container list is assembled from four independent sources on the
// InstrumentationConfig. A container that appears in only one of them must still show
// up, and the merge must key on the container name rather than duplicate it.
func TestPopulateWorkloadFields_ContainersAreMergedFromEverySectionOfTheInstrumentationConfig(t *testing.T) {
	f := wpInstrumentedFixture()
	ic := wpIC("inventory-service", f.containers)
	ic.Spec.WorkloadCollectorConfig = []commonapi.ContainerCollectorConfig{{ContainerName: "collector-only"}}
	ic.Spec.ContainersOverrides = append(ic.Spec.ContainersOverrides, odigosv1.ContainerOverride{ContainerName: "override-only"})
	ic.Status.RuntimeDetailsByContainer = []odigosv1.RuntimeDetailsByContainer{{ContainerName: "runtime-only"}}
	f.ic = ic

	w := wpPopulate(t, f)

	names := make([]string, 0, len(w.Containers))
	for _, c := range w.Containers {
		names = append(names, c.ContainerName)
	}
	assert.Equal(t, []string{"app", "collector-only", "override-only", "runtime-only"}, names,
		"containers must be merged by name and sorted")

	byName := map[string]*model.K8sWorkloadContainer{}
	for _, c := range w.Containers {
		byName[c.ContainerName] = c
	}
	assert.NotNil(t, byName["app"].AgentEnabled, "the Spec.Containers entry feeds agentEnabled")
	assert.NotNil(t, byName["collector-only"].CollectorConfig, "the WorkloadCollectorConfig entry feeds collectorConfig")
	assert.NotNil(t, byName["override-only"].Overrides, "the ContainersOverrides entry feeds overrides")
	assert.NotNil(t, byName["runtime-only"].RuntimeInfo, "the RuntimeDetailsByContainer entry feeds runtimeInfo")
}

// runtimeInfo.completed drives whether the UI shows the language column at all, and the
// detected-language list is only computed once detection has completed.
func TestPopulateWorkloadFields_RuntimeInfoCompletionTracksTheDetectionResults(t *testing.T) {
	f := wpInstrumentedFixture()

	w := wpPopulate(t, f)
	require.NotNil(t, w.RuntimeInfo)
	assert.False(t, w.RuntimeInfo.Completed, "no detection results means detection has not completed")
	assert.Empty(t, w.RuntimeInfo.DetectedLanguages)
	assert.Empty(t, w.RuntimeInfo.Containers)

	ic := wpIC("inventory-service", f.containers)
	ic.Status.RuntimeDetailsByContainer = []odigosv1.RuntimeDetailsByContainer{
		{ContainerName: "z-app", Language: "java"},
		{ContainerName: "a-app", Language: "python"},
	}
	f.ic = ic

	w = wpPopulate(t, f)
	require.NotNil(t, w.RuntimeInfo)
	assert.True(t, w.RuntimeInfo.Completed)
	assert.Equal(t, []model.ProgrammingLanguage{"java", "python"}, w.RuntimeInfo.DetectedLanguages)
	require.Len(t, w.RuntimeInfo.Containers, 2)
	assert.Equal(t, "a-app", w.RuntimeInfo.Containers[0].ContainerName, "runtime info containers are sorted by name")
	assert.Equal(t, "z-app", w.RuntimeInfo.Containers[1].ContainerName)
}

// The installation-level ignoredContainers list has to reach the language collection.
// A sidecar odigos never instruments must not make the workload look like a polyglot
// service in the UI's language filter — but it is still a real container, so it keeps
// its row in the runtime-info table.
func TestPopulateWorkloadFields_AnIgnoredContainersLanguageIsNotReported(t *testing.T) {
	f := wpInstrumentedFixture()
	ic := wpIC("inventory-service", f.containers)
	ic.Status.RuntimeDetailsByContainer = []odigosv1.RuntimeDetailsByContainer{
		{ContainerName: "app", Language: "java"},
		{ContainerName: wpIgnoredContainer, Language: "go"},
	}
	f.ic = ic

	w := wpPopulate(t, f)

	require.NotNil(t, w.RuntimeInfo)
	assert.Equal(t, []model.ProgrammingLanguage{"java"}, w.RuntimeInfo.DetectedLanguages)
	require.Len(t, w.RuntimeInfo.Containers, 2, "the ignored container is still listed, only its language is excluded")
}

// When pod manifest injection is optional the agent is enabled without a restart, so a
// container carrying no ODIGOS_DISTRO_NAME env var is still instrumented. Dropping the
// optional-container set makes those workloads report "no agent injected" even though
// their agents are running and healthy.
func TestPopulateWorkloadFields_AnOptionalPodManifestInjectionContainerCountsAsInstrumented(t *testing.T) {
	// No distro env var on the pod: without the optional-injection set this container
	// is skipped entirely by the processes-health scan.
	containers := []wpContainer{{
		name: "app", ready: true, hasInstance: true, instanceHealthy: wpBool(true),
		instanceComponent: &odigosv1.InstrumentationLibraryStatus{
			Name: "net/http", Type: odigosv1.InstrumentationLibraryTypeInstrumentation, Healthy: wpBool(true),
		},
	}}
	ic := wpIC("inventory-service", containers)
	ic.Spec.PodManifestInjectionOptional = true
	ic.Spec.Containers[0].PodManifestInjectionOptional = true
	f := wpFixture{
		availableReplicas: 1,
		ic:                ic,
		source:            wpWorkloadSource("", false),
		containers:        containers,
	}

	w := wpPopulate(t, f)

	require.NotNil(t, w.Conditions)
	require.NotNil(t, w.Conditions.ProcessesAgentHealth)
	assert.Equal(t, model.DesiredStateProgressSuccess, w.Conditions.ProcessesAgentHealth.Status)
	require.NotNil(t, w.Conditions.ProcessesAgentHealth.ReasonEnum)
	assert.Equal(t, string(status.ProcessesHealthStatusReasonAllHealthy), *w.Conditions.ProcessesAgentHealth.ReasonEnum)
}

func TestPopulateWorkloadFields_RollbackOccurredIsReadFromTheInstrumentationConfigStatus(t *testing.T) {
	f := wpInstrumentedFixture()
	ic := wpIC("inventory-service", f.containers)
	ic.Status.RollbackOccurred = true
	f.ic = ic

	assert.True(t, wpPopulate(t, f).RollbackOccurred)

	f.ic = wpIC("inventory-service", f.containers)
	assert.False(t, wpPopulate(t, f).RollbackOccurred)
}

// The processes-agent-health condition is what turns "the agent crashed" into a red
// badge. Each of these outcomes is reached through the real pod/instance loading path.
func TestPopulateWorkloadFields_TheProcessesHealthConditionReflectsTheReportedInstances(t *testing.T) {
	unhealthyComponent := &odigosv1.InstrumentationLibraryStatus{
		Name:    "net/http",
		Type:    odigosv1.InstrumentationLibraryTypeInstrumentation,
		Healthy: wpBool(false),
		Message: "failed to load",
	}

	for _, tt := range []struct {
		name       string
		container  wpContainer
		wantStatus model.DesiredStateProgress
		wantReason status.ProcessesHealthStatusReason
	}{
		{
			name:       "a healthy instance reports success",
			container:  wpHealthyContainer("app"),
			wantStatus: model.DesiredStateProgressSuccess,
			wantReason: status.ProcessesHealthStatusReasonAllHealthy,
		},
		{
			name: "an unhealthy instance reports failure",
			container: wpContainer{
				name: "app", distro: wpReportingDistro, ready: true,
				hasInstance: true, instanceHealthy: wpBool(false), instanceMessage: "agent crashed",
			},
			wantStatus: model.DesiredStateProgressFailure,
			wantReason: status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		},
		{
			name: "a healthy instance with an unhealthy library still reports failure",
			container: wpContainer{
				name: "app", distro: wpReportingDistro, ready: true,
				hasInstance: true, instanceHealthy: wpBool(true), instanceComponent: unhealthyComponent,
			},
			wantStatus: model.DesiredStateProgressFailure,
			wantReason: status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		},
		{
			name: "an instance that has not reported health yet is starting",
			container: wpContainer{
				name: "app", distro: wpReportingDistro, ready: true, hasInstance: true,
			},
			wantStatus: model.DesiredStateProgressWaiting,
			wantReason: status.ProcessesHealthStatusReasonStarting,
		},
		{
			name: "a ready instrumented container with no instance is still waiting",
			container: wpContainer{
				name: "app", distro: wpReportingDistro, ready: true,
			},
			wantStatus: model.DesiredStateProgressWaiting,
			wantReason: status.ProcessesHealthStatusReasonNoProcesses,
		},
		{
			name: "an instrumented container that is not ready yet is waiting on the container",
			container: wpContainer{
				name: "app", distro: wpReportingDistro,
			},
			wantStatus: model.DesiredStateProgressWaiting,
			wantReason: status.ProcessesHealthStatusReasonContainersNotReady,
		},
		{
			name: "a distro that does not report instances is unsupported, not unhealthy",
			container: wpContainer{
				name: "app", distro: wpSilentDistro, ready: true,
			},
			wantStatus: model.DesiredStateProgressIrrelevant,
			wantReason: status.ProcessesHealthStatusReasonUnsupported,
		},
		{
			name:       "a container with no agent at all is irrelevant",
			container:  wpContainer{name: "app", ready: true},
			wantStatus: model.DesiredStateProgressIrrelevant,
			wantReason: status.ProcessesHealthStatusReasonNoAgentInjected,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			containers := []wpContainer{tt.container}
			f := wpFixture{
				availableReplicas: 1,
				ic:                wpIC("inventory-service", containers),
				source:            wpWorkloadSource("", false),
				containers:        containers,
			}

			w := wpPopulate(t, f)

			require.NotNil(t, w.Conditions)
			got := w.Conditions.ProcessesAgentHealth
			require.NotNil(t, got, "every instrumented workload needs a processes-health condition")
			assert.Equal(t, tt.wantStatus, got.Status)
			require.NotNil(t, got.ReasonEnum)
			assert.Equal(t, string(tt.wantReason), *got.ReasonEnum)
		})
	}
}

// The seven conditions the workload page renders side by side. A nil here is a blank
// row in the UI, and the aggregated health silently ignores it.
func TestPopulateWorkloadFields_EveryWorkloadConditionIsPopulated(t *testing.T) {
	w := wpPopulate(t, wpSteadyStateFixture())

	require.NotNil(t, w.Conditions)
	for _, tt := range []struct {
		name      string
		condition *model.DesiredConditionStatus
	}{
		{"runtimeDetection", w.Conditions.RuntimeDetection},
		{"agentInjectionEnabled", w.Conditions.AgentInjectionEnabled},
		{"rollout", w.Conditions.Rollout},
		{"podsManifestInjection", w.Conditions.PodsManifestInjection},
		{"agentInjected", w.Conditions.AgentInjected},
		{"processesAgentHealth", w.Conditions.ProcessesAgentHealth},
		{"expectingTelemetry", w.Conditions.ExpectingTelemetry},
	} {
		assert.NotNil(t, tt.condition, "condition %s must be computed", tt.name)
	}
}

// podsAgentInjectionStatus is exposed both on its own field and inside conditions. They
// are assigned from the same computation and must never drift apart.
func TestPopulateWorkloadFields_PodsAgentInjectionStatusMatchesTheAgentInjectedCondition(t *testing.T) {
	w := wpPopulate(t, wpInstrumentedFixture())

	require.NotNil(t, w.Conditions)
	require.NotNil(t, w.PodsAgentInjectionStatus)
	assert.Equal(t, w.Conditions.AgentInjected, w.PodsAgentInjectionStatus)
}

// The namespace list query deliberately skips InstrumentationConfigs and pods because
// loading them cluster-wide is what pushes the UI pod past its memory limit. If this
// helper starts populating them, the source-selection screen regresses at scale.
func TestPopulateNamespaceWorkloadLightFields_OnlyTheThreeCheapFieldsAreComputed(t *testing.T) {
	ctx := wpLoaderContext(t, wpInstrumentedFixture())
	id := wpWorkloadID()
	w := &model.K8sWorkload{ID: &id}

	populateNamespaceWorkloadLightFields(ctx, loaders.For(ctx), w)

	require.NotNil(t, w.MarkedForInstrumentation)
	require.NotNil(t, w.MarkedForInstrumentation.MarkedForInstrumentation)
	assert.True(t, *w.MarkedForInstrumentation.MarkedForInstrumentation)
	require.NotNil(t, w.DataStreamNames)
	require.NotNil(t, w.NumberOfInstances)
	assert.Equal(t, 3, *w.NumberOfInstances)

	for _, tt := range []struct {
		field string
		set   bool
	}{
		{"serviceName", w.ServiceName != nil},
		{"runtimeInfo", w.RuntimeInfo != nil},
		{"containers", w.Containers != nil},
		{"conditions", w.Conditions != nil},
		{"podsAgentInjectionStatus", w.PodsAgentInjectionStatus != nil},
		{"workloadOdigosHealthStatus", w.WorkloadOdigosHealthStatus != nil},
	} {
		assert.False(t, tt.set, "%s must stay unset: it needs an expensive cluster-wide load", tt.field)
	}
}

// The light path computes markedForInstrumentation from the same source decision as the
// full path, so the source-selection screen and the workloads list agree.
func TestPopulateNamespaceWorkloadLightFields_MarkedForInstrumentationMatchesTheFullPath(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source *odigosv1.Source
	}{
		{"an enabled source", wpWorkloadSource("", false)},
		{"a disabled source", wpWorkloadSource("", true)},
		{"a source carrying data streams", wpSourceWithDataStream("payments")},
		{"no source", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := wpInstrumentedFixture()
			f.source = tt.source

			full := wpPopulate(t, f)

			ctx := wpLoaderContext(t, f)
			id := wpWorkloadID()
			light := &model.K8sWorkload{ID: &id}
			populateNamespaceWorkloadLightFields(ctx, loaders.For(ctx), light)

			assert.Equal(t, full.MarkedForInstrumentation, light.MarkedForInstrumentation)
			assert.Equal(t, full.DataStreamNames, light.DataStreamNames)
			assert.Equal(t, full.NumberOfInstances, light.NumberOfInstances)
		})
	}
}

func wpSourceWithDataStream(streamName string) *odigosv1.Source {
	source := wpWorkloadSource("", false)
	source.Labels[k8sconsts.SourceDataStreamLabelPrefix+streamName] = "true"
	return source
}
