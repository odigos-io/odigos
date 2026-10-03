package graph

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	generatedstatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// Everything in this file drives the resolvers of workload.resolvers.go against a fake
// cluster. Fixture names are prefixed "wr" (workload resolvers) to stay clear of the other
// helper families already living in package graph.
const (
	wrOdigosNamespace = "odigos-wr-system"
	wrAppNamespace    = "wr-app"
	wrOtherNamespace  = "wr-other"

	wrWorkloadName = "wr-workload"
	wrOtherName    = "wr-other-workload"

	wrContainerName = "app"
	wrSidecarName   = "sidecar"

	// a distro that isDistroExpectingInstrumentationInstances accepts, so a container
	// carrying it is expected to report InstrumentationInstances.
	wrReportingDistro = "golang-community"
	// a real distro deliberately absent from that allow list.
	wrSilentDistro = "dotnet-community"

	wrAgentsMetaHash = "wr-agents-meta-hash"
	wrServiceName    = "wr-service"
	wrDataStream     = "wr-stream"
)

func wrBool(b bool) *bool { return &b }

func wrStr(s string) *string { return &s }

func wrKind(k model.K8sResourceKind) *model.K8sResourceKind { return &k }

// wrReplicaSetName is the owner reference name a pod of the given workload carries.
// workload.PodWorkloadObject resolves ReplicaSet to Deployment by stripping everything
// after the last hyphen, so no ReplicaSet object has to exist in the cluster.
func wrReplicaSetName(workloadName string) string { return workloadName + "-7d4c8b5f9b" }

func wrPodName(workloadName string, ordinal int) string {
	return wrReplicaSetName(workloadName) + "-" + string(rune('a'+ordinal)) + "0000"
}

// wrRuntimeObjectName is the InstrumentationConfig name and the InstrumentedAppNameLabel
// value for a Deployment of the given name.
func wrRuntimeObjectName(workloadName string) string {
	return workload.CalculateWorkloadRuntimeObjectName(workloadName, k8sconsts.WorkloadKindDeployment)
}

func wrScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, odigosv1.AddToScheme(s))
	return s
}

// wrEffectiveConfigMap is the ConfigMap Loaders.LoadConfig reads. The body must be
// non-empty YAML: an empty body unmarshals into a nil *common.OdigosConfiguration and
// loadWorkloadPods then nil-derefs on odigosConfiguration.Rollout.
func wrEffectiveConfigMap(configYaml string) *corev1.ConfigMap {
	if configYaml == "" {
		configYaml = "configVersion: 1\n"
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: wrOdigosNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: configYaml},
	}
}

func wrNamespaceObject(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

// wrDeployment carries the selector that fetchWorkloadPodsWithSelector turns into the pod
// label selector, plus the AvailableReplicas that numberOfInstances reports.
func wrDeployment(namespace, name string, availableReplicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          availableReplicas,
			ReadyReplicas:     availableReplicas,
			AvailableReplicas: availableReplicas,
			UpdatedReplicas:   availableReplicas,
		},
	}
}

// wrContainerFixture describes a pod container in the terms the resolvers branch on rather
// than in raw kubernetes terms; wrPod synthesises the manifest and container status that
// produce it.
type wrContainerFixture struct {
	name    string
	distro  string
	started bool
	ready   bool
	// waitingReason, when set to CrashLoopBackOff, is what makes IsCrashLoop true.
	waitingReason string
}

func wrHealthyContainer(name, distro string) wrContainerFixture {
	return wrContainerFixture{name: name, distro: distro, started: true, ready: true}
}

type wrPodFixture struct {
	name           string
	workloadName   string
	namespace      string
	agentsMetaHash string
	containers     []wrContainerFixture
}

func wrPod(f wrPodFixture) *corev1.Pod {
	labels := map[string]string{"app": f.workloadName}
	if f.agentsMetaHash != "" {
		labels[k8sconsts.OdigosAgentsMetaHashLabel] = f.agentsMetaHash
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              f.name,
			Namespace:         f.namespace,
			Labels:            labels,
			CreationTimestamp: metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       wrReplicaSetName(f.workloadName),
				UID:        "rs-uid",
			}},
		},
		Spec: corev1.PodSpec{NodeName: "wr-node"},
	}
	for _, c := range f.containers {
		container := corev1.Container{Name: c.name}
		if c.distro != "" {
			container.Env = []corev1.EnvVar{{Name: k8sconsts.OdigosEnvVarDistroName, Value: c.distro}}
		}
		pod.Spec.Containers = append(pod.Spec.Containers, container)

		containerStatus := corev1.ContainerStatus{
			Name:    c.name,
			Ready:   c.ready,
			Started: wrBool(c.started),
		}
		if c.waitingReason != "" {
			containerStatus.State.Waiting = &corev1.ContainerStateWaiting{
				Reason:  c.waitingReason,
				Message: "back-off restarting failed container",
			}
		} else {
			containerStatus.State.Running = &corev1.ContainerStateRunning{
				StartedAt: metav1.NewTime(time.Date(2026, 1, 2, 3, 5, 0, 0, time.UTC)),
			}
		}
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, containerStatus)
	}
	return pod
}

// wrSimplePod is the common single-container pod of the main fixture workload.
func wrSimplePod(ordinal int, containers ...wrContainerFixture) *corev1.Pod {
	return wrPod(wrPodFixture{
		name:           wrPodName(wrWorkloadName, ordinal),
		workloadName:   wrWorkloadName,
		namespace:      wrAppNamespace,
		agentsMetaHash: wrAgentsMetaHash,
		containers:     containers,
	})
}

func wrIC(namespace, workloadName string, mutators ...func(*odigosv1.InstrumentationConfig)) *odigosv1.InstrumentationConfig {
	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: wrRuntimeObjectName(workloadName), Namespace: namespace},
		Spec: odigosv1.InstrumentationConfigSpec{
			ServiceName:           wrServiceName,
			AgentInjectionEnabled: true,
			AgentsMetaHash:        wrAgentsMetaHash,
			Containers: []odigosv1.ContainerAgentConfig{{
				ContainerName:  wrContainerName,
				AgentEnabled:   true,
				OtelDistroName: wrReportingDistro,
			}},
			ContainersOverrides: []odigosv1.ContainerOverride{{ContainerName: wrContainerName}},
		},
	}
	for _, m := range mutators {
		m(ic)
	}
	return ic
}

// wrMainIC is the InstrumentationConfig of the main fixture workload.
func wrMainIC(mutators ...func(*odigosv1.InstrumentationConfig)) *odigosv1.InstrumentationConfig {
	return wrIC(wrAppNamespace, wrWorkloadName, mutators...)
}

// wrReconciledIC is the IC of a workload the control loop has already converged on: all
// four status conditions present, so CalculateRolloutStatus returns non-nil and the other
// three conditions do not report Unknown.
func wrReconciledIC(mutators ...func(*odigosv1.InstrumentationConfig)) *odigosv1.InstrumentationConfig {
	reconciled := func(ic *odigosv1.InstrumentationConfig) {
		ic.Status.RuntimeDetailsByContainer = []odigosv1.RuntimeDetailsByContainer{{
			ContainerName:  wrContainerName,
			Language:       "go",
			RuntimeVersion: "1.26.0",
		}}
		ic.Status.Conditions = []metav1.Condition{
			wrCondition(odigosv1.RuntimeDetectionStatusConditionType, string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully)),
			wrCondition(generatedstatus.AgentEnabledType, string(generatedstatus.AgentEnabledReasonEnabledSuccessfully)),
			wrCondition(generatedstatus.PodsManifestInjectionType, string(generatedstatus.PodsManifestInjectionReasonPodsAppliedSuccessfully_Enabled)),
			wrCondition(odigosv1.WorkloadRolloutStatusConditionType, string(odigosv1.WorkloadRolloutReasonRolloutFinished)),
		}
	}
	return wrMainIC(append([]func(*odigosv1.InstrumentationConfig){reconciled}, mutators...)...)
}

func wrCondition(conditionType string, reason string) metav1.Condition {
	return metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            "reconciled by the fixture",
		LastTransitionTime: metav1.NewTime(time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)),
		ObservedGeneration: 1,
	}
}

type wrInstanceFixture struct {
	name          string
	namespace     string
	workloadName  string
	podName       string
	containerName string
	pid           string
	healthy       *bool
	message       string
	components    []odigosv1.InstrumentationLibraryStatus
}

// wrInstance builds an InstrumentationInstance the way odiglet writes it: keyed by the
// owner-pod label and InstrumentedAppNameLabel, not by its own name.
func wrInstance(f wrInstanceFixture) *odigosv1.InstrumentationInstance {
	if f.pid == "" {
		f.pid = "1"
	}
	return &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.name,
			Namespace: f.namespace,
			Labels: map[string]string{
				odigosv1.OwnerPodNameLabel:      f.podName,
				consts.InstrumentedAppNameLabel: wrRuntimeObjectName(f.workloadName),
			},
		},
		Spec: odigosv1.InstrumentationInstanceSpec{ContainerName: f.containerName},
		Status: odigosv1.InstrumentationInstanceStatus{
			Healthy:    f.healthy,
			Message:    f.message,
			Components: f.components,
			IdentifyingAttributes: []odigosv1.Attribute{
				{Key: processAttributeNamePid, Value: f.pid},
				{Key: "process.executable.name", Value: "app"},
			},
		},
	}
}

// wrHealthyInstance is an instance of the main fixture workload reporting one healthy
// instrumentation library.
func wrHealthyInstance(podName, containerName, pid string) *odigosv1.InstrumentationInstance {
	return wrInstance(wrInstanceFixture{
		name:          "inst-" + podName + "-" + containerName,
		namespace:     wrAppNamespace,
		workloadName:  wrWorkloadName,
		podName:       podName,
		containerName: containerName,
		pid:           pid,
		healthy:       wrBool(true),
		components:    []odigosv1.InstrumentationLibraryStatus{wrComponent("net/http", wrBool(true))},
	})
}

func wrComponent(name string, healthy *bool) odigosv1.InstrumentationLibraryStatus {
	return odigosv1.InstrumentationLibraryStatus{
		Name:    name,
		Type:    odigosv1.InstrumentationLibraryTypeInstrumentation,
		Healthy: healthy,
	}
}

// wrSource builds the workload Source CR that fetchSourcesForWorkload matches by label.
func wrSource(namespace, workloadName string, disabled bool, dataStreams ...string) *odigosv1.Source {
	labels := map[string]string{
		k8sconsts.WorkloadNamespaceLabel: namespace,
		k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindDeployment),
		k8sconsts.WorkloadNameLabel:      workloadName,
	}
	for _, ds := range dataStreams {
		labels[k8sconsts.SourceDataStreamLabelPrefix+ds] = "true"
	}
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: workloadName + "-source", Namespace: namespace, Labels: labels},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      workloadName,
				Namespace: namespace,
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
			DisableInstrumentation: disabled,
		},
	}
}

// wrNamespaceSource builds the namespace-scoped Source CR, which is what
// k8sNamespaceResolver.MarkedForInstrumentation and DataStreamNames read.
func wrNamespaceSource(namespace string, disabled bool, dataStreams ...string) *odigosv1.Source {
	labels := map[string]string{
		k8sconsts.WorkloadNamespaceLabel: namespace,
		k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
		k8sconsts.WorkloadNameLabel:      namespace,
	}
	for _, ds := range dataStreams {
		labels[k8sconsts.SourceDataStreamLabelPrefix+ds] = "true"
	}
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: namespace + "-ns-source", Namespace: namespace, Labels: labels},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      namespace,
				Namespace: namespace,
				Kind:      k8sconsts.WorkloadKindNamespace,
			},
			DisableInstrumentation: disabled,
		},
	}
}

func wrWorkloadID() model.K8sWorkloadID {
	return model.K8sWorkloadID{
		Namespace: wrAppNamespace,
		Kind:      model.K8sResourceKindDeployment,
		Name:      wrWorkloadName,
	}
}

// wrSingleWorkloadFilter is the filter the workload detail page issues.
func wrSingleWorkloadFilter() *model.WorkloadFilter {
	return &model.WorkloadFilter{
		Namespace: wrStr(wrAppNamespace),
		Kind:      wrKind(model.K8sResourceKindDeployment),
		Name:      wrStr(wrWorkloadName),
	}
}

// wrHarness seeds a fake cluster, primes the loaders for the single fixture workload and
// returns a context carrying them plus a Resolver wired to the same client.
//
// The zero-value OdigosMetricsConsumer is deliberate: it reports (zero, false) from
// GetSingleSourceMetrics without starting any goroutine. Populating it for real needs
// RunDeleteWatcherAndNotifications, whose client-go RetryWatcher goroutine reads the
// non-atomic kube.DefaultClient package global that other tests in this package write.
func wrHarness(t *testing.T, objects ...client.Object) (context.Context, *Resolver) {
	t.Helper()
	return wrHarnessWithFilter(t, wrSingleWorkloadFilter(), objects...)
}

func wrHarnessWithFilter(t *testing.T, filter *model.WorkloadFilter, objects ...client.Object) (context.Context, *Resolver) {
	t.Helper()
	ctx, r, l := wrUnloadedHarness(t, wrInterceptors{}, objects...)
	require.NoError(t, l.LoadWorkloadsWithFilter(ctx, filter))
	return ctx, r
}

// wrInterceptors makes one kind of List fail, so that a cluster read error can be observed
// downstream of the priming step. Priming a single-workload filter Gets the workload and
// its InstrumentationConfig and Lists Sources, so failing the pod or instance List reaches
// only the resolver under test.
type wrInterceptors struct {
	failPodList      bool
	failInstanceList bool
	failSourceList   bool
}

func (i wrInterceptors) funcs() interceptor.Funcs {
	if !i.failPodList && !i.failInstanceList && !i.failSourceList {
		return interceptor.Funcs{}
	}
	return interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			switch list.(type) {
			case *corev1.PodList:
				if i.failPodList {
					return apierrors.NewServiceUnavailable("the informer cache is not synced")
				}
			case *odigosv1.InstrumentationInstanceList:
				if i.failInstanceList {
					return apierrors.NewServiceUnavailable("the informer cache is not synced")
				}
			case *odigosv1.SourceList:
				if i.failSourceList {
					return apierrors.NewServiceUnavailable("the informer cache is not synced")
				}
			}
			return c.List(ctx, list, opts...)
		},
	}
}

// wrFailingHarness primes the loaders for the single fixture workload with one kind of
// cluster read broken.
func wrFailingHarness(t *testing.T, interceptors wrInterceptors, objects ...client.Object) (context.Context, *Resolver) {
	t.Helper()
	ctx, r, l := wrUnloadedHarness(t, interceptors, objects...)
	require.NoError(t, l.LoadWorkloadsWithFilter(ctx, wrSingleWorkloadFilter()))
	return ctx, r
}

// wrUnloadedHarness returns the loaders without priming them, for the query resolvers that
// do their own loading.
func wrUnloadedHarness(t *testing.T, interceptors wrInterceptors, objects ...client.Object) (context.Context, *Resolver, *loaders.Loaders) {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, wrOdigosNamespace)
	wrSetTier(t, model.TierOnprem)

	hasEffectiveConfig := false
	for _, o := range objects {
		if cm, ok := o.(*corev1.ConfigMap); ok && cm.Name == consts.OdigosEffectiveConfigName {
			hasEffectiveConfig = true
		}
	}
	if !hasEffectiveConfig {
		objects = append(objects, wrEffectiveConfigMap(""))
	}

	c := fake.NewClientBuilder().WithScheme(wrScheme(t)).WithObjects(objects...).
		WithInterceptorFuncs(interceptors.funcs()).Build()
	l := loaders.NewLoaders(logr.Discard(), c)
	ctx := loaders.WithLoaders(t.Context(), l)
	return ctx, &Resolver{
		Logger:          logr.Discard(),
		K8sCacheClient:  c,
		MetricsConsumer: &collectormetrics.OdigosMetricsConsumer{},
	}, l
}

// wrSetTier points the kube.DefaultClient package global at a typed fake clientset holding
// the odigos-deployment ConfigMap, which is where services.GetTier reads the tier from.
// Restoring it in cleanup is safe here because nothing in this file starts a goroutine
// that reads the global.
func wrSetTier(t *testing.T, tier model.Tier) {
	t.Helper()
	clientset := k8sfake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosDeploymentConfigMapName,
			Namespace: wrOdigosNamespace,
		},
		Data: map[string]string{k8sconsts.OdigosDeploymentConfigMapTierKey: string(tier)},
	})
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{Interface: clientset, DynamicClient: wrDynamicClient()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })
}

// wrDynamicClient stands in for the dynamic client kube.IsDeploymentConfigAvailable probes.
// Listing DeploymentConfigs fails, which is what a non-OpenShift cluster reports and what
// keeps the multi-workload manifest fetch on its plain-kubernetes path.
func wrDynamicClient() *dynamicfake.FakeDynamicClient {
	gvr := schema.GroupVersionResource{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}
	c := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "DeploymentConfigList"})
	c.PrependReactor("list", "deploymentconfigs", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(gvr.GroupResource(), "")
	})
	return c
}

// wrExecute runs a real GraphQL operation through the generated executable schema, so the
// graphql.FieldContext parent chain the resolvers walk is the one gqlgen actually builds
// rather than one assembled by the test.
func wrExecute(t *testing.T, ctx context.Context, r *Resolver, query string, variables map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, errs := wrExecuteRaw(t, ctx, r, query, variables)
	require.Empty(t, errs, "graphql operation returned errors")
	return data
}

func wrExecuteRaw(t *testing.T, ctx context.Context, r *Resolver, query string, variables map[string]interface{}) (map[string]interface{}, []string) {
	t.Helper()
	exec := executor.New(NewExecutableSchema(Config{Resolvers: r}))
	ctx = graphql.StartOperationTrace(ctx)
	opCtx, gqlErrs := exec.CreateOperationContext(ctx, &graphql.RawParams{Query: query, Variables: variables})
	require.Empty(t, gqlErrs, "failed to build the graphql operation context")

	responseHandler, ctx := exec.DispatchOperation(ctx, opCtx)
	res := responseHandler(ctx)
	require.NotNil(t, res)

	messages := make([]string, 0, len(res.Errors))
	for _, e := range res.Errors {
		messages = append(messages, e.Message)
	}

	var data map[string]interface{}
	if len(res.Data) > 0 {
		require.NoError(t, json.Unmarshal(res.Data, &data))
	}
	return data, messages
}

// wrSingleWorkloadVars are the variables for a `workloads(filter: $filter)` operation
// scoped to the main fixture workload.
func wrSingleWorkloadVars() map[string]interface{} {
	return map[string]interface{}{"filter": map[string]interface{}{
		"namespace": wrAppNamespace,
		"kind":      string(model.K8sResourceKindDeployment),
		"name":      wrWorkloadName,
	}}
}

// wrOnlyWorkload digs the single workload out of a `workloads(filter: ...)` response.
func wrOnlyWorkload(t *testing.T, data map[string]interface{}) map[string]interface{} {
	t.Helper()
	list := wrList(t, data, "workloads")
	require.Len(t, list, 1)
	return wrObject(t, list[0])
}

func wrList(t *testing.T, parent map[string]interface{}, field string) []interface{} {
	t.Helper()
	raw, ok := parent[field]
	require.True(t, ok, "field %q missing from %v", field, parent)
	list, ok := raw.([]interface{})
	require.True(t, ok, "field %q is not a list: %v", field, raw)
	return list
}

func wrObject(t *testing.T, raw interface{}) map[string]interface{} {
	t.Helper()
	obj, ok := raw.(map[string]interface{})
	require.True(t, ok, "not an object: %v", raw)
	return obj
}

func wrChild(t *testing.T, parent map[string]interface{}, field string) map[string]interface{} {
	t.Helper()
	raw, ok := parent[field]
	require.True(t, ok, "field %q missing from %v", field, parent)
	return wrObject(t, raw)
}

func wrString(t *testing.T, parent map[string]interface{}, field string) string {
	t.Helper()
	raw, ok := parent[field]
	require.True(t, ok, "field %q missing from %v", field, parent)
	s, ok := raw.(string)
	require.True(t, ok, "field %q is not a string: %v", field, raw)
	return s
}
