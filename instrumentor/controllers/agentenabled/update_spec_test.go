package agentenabled

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	agentInjectionEnabled "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ****************
// updateInstrumentationConfigSpec() tests
// ****************

const (
	goDistroName  = "golang-community"
	obiDistroName = "opentelemetry-ebpf-instrumentation"
)

// specSyncEnv holds the inputs updateInstrumentationConfigSpec needs, and the cluster objects
// that must exist for a workload to be considered ready for instrumentation.
type specSyncEnv struct {
	setup      *syncTestSetup
	ctx        context.Context
	provider   *distros.Provider
	conf       *common.OdigosConfiguration
	deployment *appsv1.Deployment
	pw         k8sconsts.PodWorkload
	objects    []client.Object
}

func newSpecSyncEnv(t *testing.T) *specSyncEnv {
	t.Helper()

	setup := newSyncTestSetup()

	getter, err := distros.NewCommunityGetter()
	require.NoError(t, err)
	provider, err := distros.NewProvider(distros.NewCommunityDefaulter(), getter)
	require.NoError(t, err)

	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	injectionMethod := common.PodManifestEnvInjectionMethod

	return &specSyncEnv{
		setup: setup,
		// updateInstrumentationConfigSpec logs via commonlogger.FromContext
		ctx:        logr.NewContext(setup.ctx, logr.Discard()),
		provider:   provider,
		conf:       &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &injectionMethod},
		deployment: deployment,
		pw:         testutil.PodWorkloadFromDeployment(deployment),
		objects:    []client.Object{setup.ns, deployment, newReadyNodeCollectorsGroup()},
	}
}

// syncWith runs the function under test against a client holding the base cluster objects
// (namespace, workload, ready node collectors group) plus any extra objects.
func (e *specSyncEnv) syncWith(ic *odigosv1alpha1.InstrumentationConfig, extraObjects ...client.Object) (*agentInjectedStatusCondition, error) {
	objects := append(append([]client.Object{}, e.objects...), extraObjects...)
	c := e.setup.newFakeClient(objects...)
	return updateInstrumentationConfigSpec(e.ctx, c, e.pw, ic, e.provider, e.conf)
}

// newReadyNodeCollectorsGroup returns a node collectors group that satisfies the instrumentation
// prerequisites: ready, with an otlp receiver for traces.
func newReadyNodeCollectorsGroup() *odigosv1alpha1.CollectorsGroup {
	return &odigosv1alpha1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: odigosv1alpha1.CollectorsGroupSpec{
			Role: odigosv1alpha1.CollectorsGroupRoleNodeCollector,
		},
		Status: odigosv1alpha1.CollectorsGroupStatus{
			Ready:           true,
			ReceiverSignals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}
}

// newSpecSyncIC builds an InstrumentationConfig for the given runtime details.
// The containers are also listed as overrides, since the reconcile iterates the overrides
// to find the containers of the workload.
func newSpecSyncIC(deployment *appsv1.Deployment, runtimeDetails ...odigosv1alpha1.RuntimeDetailsByContainer) *odigosv1alpha1.InstrumentationConfig {
	ic := testutil.NewMockInstrumentationConfig(deployment)
	ic.Status.RuntimeDetailsByContainer = runtimeDetails
	ic.Spec.ContainersOverrides = nil
	for _, details := range runtimeDetails {
		ic.Spec.ContainersOverrides = append(ic.Spec.ContainersOverrides, odigosv1alpha1.ContainerOverride{
			ContainerName: details.ContainerName,
		})
	}
	return ic
}

func goRuntimeDetails(containerName string) odigosv1alpha1.RuntimeDetailsByContainer {
	return odigosv1alpha1.RuntimeDetailsByContainer{
		ContainerName:  containerName,
		Language:       common.GoProgrammingLanguage,
		RuntimeVersion: "1.22.0",
	}
}

func unknownLanguageRuntimeDetails(containerName string) odigosv1alpha1.RuntimeDetailsByContainer {
	return odigosv1alpha1.RuntimeDetailsByContainer{
		ContainerName: containerName,
		Language:      common.UnknownProgrammingLanguage,
	}
}

func containerAgentConfigByName(t *testing.T, ic *odigosv1alpha1.InstrumentationConfig, containerName string) odigosv1alpha1.ContainerAgentConfig {
	t.Helper()
	for _, containerConfig := range ic.Spec.Containers {
		if containerConfig.ContainerName == containerName {
			return containerConfig
		}
	}
	require.FailNowf(t, "container config not found", "no container agent config for container %s", containerName)
	return odigosv1alpha1.ContainerAgentConfig{}
}

func TestUpdateInstrumentationConfigSpec_EnablesAgentForSupportedContainer(t *testing.T) {
	// Arrange: a single go container with runtime details, and a ready node collector
	env := newSpecSyncEnv(t)
	ic := newSpecSyncIC(env.deployment, goRuntimeDetails("test"))

	// Act
	condition, err := env.syncWith(ic)

	// Assert: agent is enabled for the container and the workload is marked for pod processing
	require.NoError(t, err)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonEnabledSuccessfully), condition.Reason)
	assert.Contains(t, condition.Message, "test")
	assert.True(t, ic.Spec.AgentInjectionEnabled)

	require.Len(t, ic.Spec.Containers, 1)
	containerConfig := ic.Spec.Containers[0]
	assert.True(t, containerConfig.AgentEnabled)
	assert.Equal(t, goDistroName, containerConfig.OtelDistroName)
	// the go agent is loaded via eBPF, but the pod manifest still carries the odigos device
	assert.False(t, containerConfig.PodManifestInjectionOptional)
	assert.False(t, ic.Spec.PodManifestInjectionOptional)

	assert.NotEmpty(t, ic.Spec.AgentsMetaHash, "expected an agents meta hash for an instrumented workload")
	assert.NotNil(t, ic.Spec.AgentsMetaHashChangedTime)
}

// The agents meta hash drives workload rollouts, so recomputing the same container configs must
// not report a new hash change time - otherwise every reconcile would restart every workload.
func TestUpdateInstrumentationConfigSpec_HashChangeTimeStableAcrossReconciles(t *testing.T) {
	env := newSpecSyncEnv(t)
	ic := newSpecSyncIC(env.deployment, goRuntimeDetails("test"))

	_, err := env.syncWith(ic)
	require.NoError(t, err)
	firstHash := ic.Spec.AgentsMetaHash
	require.NotEmpty(t, firstHash)

	// backdate the recorded change time, so a rewrite of it is observable
	changedTime := metav1.NewTime(time.Now().Add(-time.Hour))
	ic.Spec.AgentsMetaHashChangedTime = &changedTime

	_, err = env.syncWith(ic)
	require.NoError(t, err)

	assert.Equal(t, firstHash, ic.Spec.AgentsMetaHash)
	assert.True(t, ic.Spec.AgentsMetaHashChangedTime.Equal(&changedTime),
		"expected the hash change time to be left untouched when the agents meta hash did not change")
}

func TestUpdateInstrumentationConfigSpec_HashChangesWhenContainersChange(t *testing.T) {
	env := newSpecSyncEnv(t)
	ic := newSpecSyncIC(env.deployment, goRuntimeDetails("test"))

	_, err := env.syncWith(ic)
	require.NoError(t, err)
	firstHash := ic.Spec.AgentsMetaHash
	require.NotEmpty(t, firstHash)

	changedTime := metav1.NewTime(time.Now().Add(-time.Hour))
	ic.Spec.AgentsMetaHashChangedTime = &changedTime

	// a second container joined the workload
	sidecar := goRuntimeDetails("sidecar")
	ic.Status.RuntimeDetailsByContainer = append(ic.Status.RuntimeDetailsByContainer, sidecar)
	ic.Spec.ContainersOverrides = append(ic.Spec.ContainersOverrides, odigosv1alpha1.ContainerOverride{
		ContainerName: sidecar.ContainerName,
	})

	_, err = env.syncWith(ic)
	require.NoError(t, err)

	assert.NotEqual(t, firstHash, ic.Spec.AgentsMetaHash, "expected a new agents meta hash after the container configs changed")
	assert.True(t, ic.Spec.AgentsMetaHashChangedTime.After(changedTime.Time),
		"expected the hash change time to be updated when the agents meta hash changed")
}

// The aggregated condition reported to the user is the most specific reason across all containers.
// Container configs are computed from a map, so the aggregation must not depend on iteration order.
func TestUpdateInstrumentationConfigSpec_AggregatedReasonIsHighestPriority(t *testing.T) {
	env := newSpecSyncEnv(t)
	env.conf.IgnoredContainers = []string{"ignored"}

	// repeat, since a single iteration could pick the expected reason by chance
	for i := 0; i < 20; i++ {
		ic := newSpecSyncIC(env.deployment,
			unknownLanguageRuntimeDetails("unsupported"),
			goRuntimeDetails("ignored"),
		)

		condition, err := env.syncWith(ic)
		require.NoError(t, err)
		require.NotNil(t, condition)

		// UnsupportedProgrammingLanguage has a higher priority than IgnoredContainer
		assert.Equal(t, odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonUnsupportedProgrammingLanguage), condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)

		assert.False(t, ic.Spec.AgentInjectionEnabled)
		assert.False(t, ic.Spec.PodManifestInjectionOptional)
		assert.Empty(t, ic.Spec.AgentsMetaHash, "expected no agents meta hash when no container is instrumented")
		assert.False(t, containerAgentConfigByName(t, ic, "ignored").AgentEnabled)
		assert.False(t, containerAgentConfigByName(t, ic, "unsupported").AgentEnabled)
	}
}

func TestUpdateInstrumentationConfigSpec_PodManifestInjectionOptional(t *testing.T) {
	t.Run("optional when the only agent does not need the pod manifest", func(t *testing.T) {
		env := newSpecSyncEnv(t)
		ic := newSpecSyncIC(env.deployment, goRuntimeDetails("test"))
		obiDistro := obiDistroName
		ic.Spec.ContainersOverrides[0].OtelDistroName = &obiDistro

		condition, err := env.syncWith(ic)

		require.NoError(t, err)
		assert.Equal(t, odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonEnabledSuccessfully), condition.Reason)
		require.Len(t, ic.Spec.Containers, 1)
		assert.Equal(t, obiDistroName, ic.Spec.Containers[0].OtelDistroName)
		assert.True(t, ic.Spec.Containers[0].PodManifestInjectionOptional)
		assert.True(t, ic.Spec.PodManifestInjectionOptional)
	})

	t.Run("required when any instrumented container needs the pod manifest", func(t *testing.T) {
		env := newSpecSyncEnv(t)
		ic := newSpecSyncIC(env.deployment,
			goRuntimeDetails("test"),
			goRuntimeDetails("sidecar"),
		)
		obiDistro := obiDistroName
		ic.Spec.ContainersOverrides[0].OtelDistroName = &obiDistro

		_, err := env.syncWith(ic)

		require.NoError(t, err)
		assert.True(t, containerAgentConfigByName(t, ic, "test").PodManifestInjectionOptional)
		assert.False(t, containerAgentConfigByName(t, ic, "sidecar").PodManifestInjectionOptional)
		assert.False(t, ic.Spec.PodManifestInjectionOptional,
			"a single container that needs the pod manifest makes it required for the whole pod")
	})
}

// After a rollback, instrumentation must stay off for the workload, but the distro decision is kept
// so the reason reported to the user (and the recovery flow) still knows what was rolled back.
func TestUpdateInstrumentationConfigSpec_RollbackDisablesAgent(t *testing.T) {
	backoffReasonFromConditions := func(reason odigosv1alpha1.AgentEnabledReason) []metav1.Condition {
		return []metav1.Condition{
			{
				Type:               agentInjectionEnabled.AgentEnabledType,
				Status:             metav1.ConditionFalse,
				Reason:             string(reason),
				LastTransitionTime: metav1.Now(),
			},
		}
	}

	testCases := []struct {
		name              string
		conditions        []metav1.Condition
		existingContainer *odigosv1alpha1.ContainerAgentConfig
		expectedReason    odigosv1alpha1.AgentEnabledReason
	}{
		{
			name:           "backoff reason from the agent enabled condition",
			conditions:     backoffReasonFromConditions(odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonImagePullBackOff)),
			expectedReason: odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonImagePullBackOff),
		},
		{
			name: "backoff reason from the existing container config",
			existingContainer: &odigosv1alpha1.ContainerAgentConfig{
				ContainerName:      "test",
				AgentEnabledReason: odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonImagePullBackOff),
			},
			expectedReason: odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonImagePullBackOff),
		},
		{
			name:           "crash loop is assumed when no backoff reason was recorded",
			expectedReason: odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonCrashLoopBackOff),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := newSpecSyncEnv(t)
			ic := newSpecSyncIC(env.deployment, goRuntimeDetails("test"))
			ic.Status.RollbackOccurred = true
			ic.Status.Conditions = tc.conditions
			if tc.existingContainer != nil {
				ic.Spec.Containers = []odigosv1alpha1.ContainerAgentConfig{*tc.existingContainer}
			}
			ic.Spec.AgentInjectionEnabled = true
			ic.Spec.AgentsMetaHash = "hash-from-before-the-rollback"

			condition, err := env.syncWith(ic)

			require.NoError(t, err)
			require.NotNil(t, condition)
			assert.Equal(t, tc.expectedReason, condition.Reason)
			assert.Equal(t, metav1.ConditionFalse, condition.Status)

			require.Len(t, ic.Spec.Containers, 1)
			containerConfig := ic.Spec.Containers[0]
			assert.False(t, containerConfig.AgentEnabled)
			assert.Equal(t, tc.expectedReason, containerConfig.AgentEnabledReason)
			assert.Equal(t, goDistroName, containerConfig.OtelDistroName, "expected the resolved distro to be preserved after a rollback")

			assert.False(t, ic.Spec.AgentInjectionEnabled)
			assert.False(t, ic.Spec.PodManifestInjectionOptional)
			assert.Empty(t, ic.Spec.AgentsMetaHash, "expected the agents meta hash to be cleared after a rollback")
		})
	}
}

// While a prerequisite is missing the workload must look completely uninstrumented, otherwise the
// pods webhook keeps injecting agents based on a stale spec.
func TestUpdateInstrumentationConfigSpec_ResetsSpecWhileWaitingForNodeCollector(t *testing.T) {
	env := newSpecSyncEnv(t)
	env.objects = []client.Object{env.setup.ns, env.deployment}
	ic := newSpecSyncIC(env.deployment, goRuntimeDetails("test"))
	ic.Spec.AgentInjectionEnabled = true
	ic.Spec.PodManifestInjectionOptional = true
	ic.Spec.AgentsMetaHash = "hash-from-when-the-collector-was-ready"
	ic.Spec.Containers = []odigosv1alpha1.ContainerAgentConfig{
		{ContainerName: "test", AgentEnabled: true, OtelDistroName: goDistroName},
	}

	condition, err := env.syncWith(ic)

	require.NoError(t, err)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
	assert.Equal(t, odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonWaitingForNodeCollector), condition.Reason)

	assert.False(t, ic.Spec.AgentInjectionEnabled)
	assert.Empty(t, ic.Spec.Containers)
	assert.Empty(t, ic.Spec.AgentsMetaHash)
	assert.NotNil(t, ic.Spec.AgentsMetaHashChangedTime, "expected the hash change time to be recorded when the hash is cleared")
}

// A workload that is already crashing before instrumentation is left alone: the spec is not
// recomputed, and the rollout condition explains why.
func TestUpdateInstrumentationConfigSpec_BackoffBeforeInstrumentation(t *testing.T) {
	env := newSpecSyncEnv(t)
	crashingPod := newCrashLoopBackOffPodWithoutOdigosLabel(env.setup.ns, env.deployment.Name, "crashing-pod")
	ic := newSpecSyncIC(env.deployment, goRuntimeDetails("test"))
	ic.Spec.AgentInjectionEnabled = true
	ic.Spec.AgentsMetaHash = "hash-from-before-the-crash"

	condition, err := env.syncWith(ic, crashingPod)

	require.NoError(t, err)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Equal(t, odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonCrashLoopBackOff), condition.Reason)

	rolloutCondition := icCondition(t, ic, odigosv1alpha1.WorkloadRolloutStatusConditionType)
	assert.Equal(t, metav1.ConditionFalse, rolloutCondition.Status)
	assert.Equal(t, string(agentInjectionEnabled.AgentEnabledReasonCrashLoopBackOff), rolloutCondition.Reason)

	assert.Empty(t, ic.Spec.Containers, "expected the container configs to be left untouched")
	assert.Equal(t, "hash-from-before-the-crash", ic.Spec.AgentsMetaHash)
}

func icCondition(t *testing.T, ic *odigosv1alpha1.InstrumentationConfig, conditionType string) metav1.Condition {
	t.Helper()
	for _, condition := range ic.Status.Conditions {
		if condition.Type == conditionType {
			return condition
		}
	}
	require.FailNowf(t, "condition not found", "no %s condition on the instrumentation config", conditionType)
	return metav1.Condition{}
}
