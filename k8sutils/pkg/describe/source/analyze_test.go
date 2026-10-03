package source

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

func sourceObject(name string, disableInstrumentation bool) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "shop"},
		Spec: odigosv1.SourceSpec{
			DisableInstrumentation: disableInstrumentation,
		},
	}
}

func analyzeTestPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "shop"},
		Spec: corev1.PodSpec{
			NodeName:   "node-1",
			Containers: []corev1.Container{{Name: "checkout"}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func ownedInstrumentationInstance(podName string, containerName string) odigosv1.InstrumentationInstance {
	return odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName + "-" + containerName,
			Namespace: "shop",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "Pod", Name: podName},
			},
		},
		Spec: odigosv1.InstrumentationInstanceSpec{ContainerName: containerName},
	}
}

func TestAnalyzeInstrumentationBySources(t *testing.T) {
	testCases := []struct {
		name                 string
		sources              *odigosv1.WorkloadSources
		expectedInstrumented bool
		expectedWorkload     string
		expectedNamespace    string
		expectedText         string
	}{
		{
			name:                 "no sources at all",
			sources:              &odigosv1.WorkloadSources{},
			expectedInstrumented: false,
			expectedWorkload:     "unset",
			expectedNamespace:    "unset",
			expectedText:         "Workload is NOT instrumented because neither the workload source nor the namespace source are present",
		},
		{
			name:                 "enabled workload source",
			sources:              &odigosv1.WorkloadSources{Workload: sourceObject("workload-source", false)},
			expectedInstrumented: true,
			expectedWorkload:     "instrumented",
			expectedNamespace:    "unset",
			expectedText:         "Workload is instrumented because the workload source is present and enabled",
		},
		{
			name:                 "disabled workload source",
			sources:              &odigosv1.WorkloadSources{Workload: sourceObject("workload-source", true)},
			expectedInstrumented: false,
			expectedWorkload:     "unset",
			expectedNamespace:    "unset",
			expectedText:         "Workload is NOT instrumented because the workload source is present and disabled",
		},
		{
			name:                 "enabled namespace source",
			sources:              &odigosv1.WorkloadSources{Namespace: sourceObject("namespace-source", false)},
			expectedInstrumented: true,
			expectedWorkload:     "unset",
			expectedNamespace:    "instrumented",
			expectedText:         "Workload is instrumented because the workload source is not present, but the namespace source is present and enabled",
		},
		{
			name:                 "disabled namespace source",
			sources:              &odigosv1.WorkloadSources{Namespace: sourceObject("namespace-source", true)},
			expectedInstrumented: false,
			expectedWorkload:     "unset",
			expectedNamespace:    "unset",
			expectedText:         "Workload is NOT instrumented because the workload source is not present, but the namespace source is present and disabled",
		},
		{
			// a disabled workload source excludes the workload from an instrumented namespace
			name: "disabled workload source overrides an enabled namespace source",
			sources: &odigosv1.WorkloadSources{
				Workload:  sourceObject("workload-source", true),
				Namespace: sourceObject("namespace-source", false),
			},
			expectedInstrumented: false,
			expectedWorkload:     "unset",
			expectedNamespace:    "instrumented",
			expectedText:         "Workload is NOT instrumented because the workload source is present and disabled",
		},
		{
			// an enabled workload source instruments the workload even inside a disabled namespace
			name: "enabled workload source overrides a disabled namespace source",
			sources: &odigosv1.WorkloadSources{
				Workload:  sourceObject("workload-source", false),
				Namespace: sourceObject("namespace-source", true),
			},
			expectedInstrumented: true,
			expectedWorkload:     "instrumented",
			expectedNamespace:    "unset",
			expectedText:         "Workload is instrumented because the workload source is present and enabled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analyze, instrumented := analyzeInstrumentationBySources(tc.sources)

			assert.Equal(t, tc.expectedInstrumented, instrumented)
			assert.Equal(t, tc.expectedInstrumented, analyze.Instrumented.Value)
			require.NotNil(t, analyze.Workload)
			assert.Equal(t, tc.expectedWorkload, analyze.Workload.Value)
			require.NotNil(t, analyze.Namespace)
			assert.Equal(t, tc.expectedNamespace, analyze.Namespace.Value)
			assert.Equal(t, tc.expectedText, analyze.InstrumentedText.Value)
		})
	}
}

func TestAnalyzeEnabledAgents(t *testing.T) {
	t.Run("instrumented workload without an instrumentation config is transitioning", func(t *testing.T) {
		analyze := analyzeEnabledAgents(&OdigosSourceResources{}, true)

		assert.Equal(t, "not created", analyze.Created.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, analyze.Created.Status)
		assert.Nil(t, analyze.CreateTime)
		assert.Empty(t, analyze.Containers)
	})

	t.Run("uninstrumented workload without an instrumentation config is settled", func(t *testing.T) {
		analyze := analyzeEnabledAgents(&OdigosSourceResources{}, false)

		assert.Equal(t, "not created", analyze.Created.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.Created.Status)
	})

	// a leftover instrumentation config for a workload that is no longer a source
	t.Run("uninstrumented workload with an instrumentation config is transitioning", func(t *testing.T) {
		resources := &OdigosSourceResources{InstrumentationConfig: &odigosv1.InstrumentationConfig{}}

		analyze := analyzeEnabledAgents(resources, false)

		assert.Equal(t, "created", analyze.Created.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, analyze.Created.Status)
	})

	t.Run("instrumented workload with an instrumentation config reports its containers", func(t *testing.T) {
		creationTime := metav1.NewTime(metav1.Now().Rfc3339Copy().Time)
		resources := &OdigosSourceResources{
			InstrumentationConfig: &odigosv1.InstrumentationConfig{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: creationTime},
				Spec: odigosv1.InstrumentationConfigSpec{
					Containers: []odigosv1.ContainerAgentConfig{
						{ContainerName: "checkout", AgentEnabled: true, OtelDistroName: "golang-community"},
					},
				},
			},
		}

		analyze := analyzeEnabledAgents(resources, true)

		assert.Equal(t, "created", analyze.Created.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.Created.Status)
		require.NotNil(t, analyze.CreateTime)
		assert.Equal(t, creationTime.String(), analyze.CreateTime.Value)
		require.Len(t, analyze.Containers, 1)
		assert.Equal(t, "checkout", analyze.Containers[0].ContainerName.Value)
	})
}

func TestAnalyzeContainersConfig(t *testing.T) {
	t.Run("optional properties are omitted when empty", func(t *testing.T) {
		containers := []odigosv1.ContainerAgentConfig{{ContainerName: "checkout", AgentEnabled: false}}

		analyze := analyzeContainersConfig(&containers)

		require.Len(t, analyze, 1)
		assert.Equal(t, "checkout", analyze[0].ContainerName.Value)
		assert.True(t, analyze[0].ContainerName.ListKey)
		assert.Equal(t, false, analyze[0].AgentEnabled.Value)
		assert.Nil(t, analyze[0].Reason)
		assert.Nil(t, analyze[0].Message)
		assert.Nil(t, analyze[0].OtelDistroName)
	})

	t.Run("every container is reported with its own decision", func(t *testing.T) {
		containers := []odigosv1.ContainerAgentConfig{
			{
				ContainerName:  "checkout",
				AgentEnabled:   true,
				OtelDistroName: "golang-community",
			},
			{
				ContainerName:       "sidecar",
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedProgrammingLanguage,
				AgentEnabledMessage: "language not supported",
			},
		}

		analyze := analyzeContainersConfig(&containers)

		require.Len(t, analyze, 2)
		assert.Equal(t, "checkout", analyze[0].ContainerName.Value)
		assert.Equal(t, true, analyze[0].AgentEnabled.Value)
		require.NotNil(t, analyze[0].OtelDistroName)
		assert.Equal(t, "golang-community", analyze[0].OtelDistroName.Value)
		assert.Nil(t, analyze[0].Reason)

		assert.Equal(t, "sidecar", analyze[1].ContainerName.Value)
		assert.Equal(t, false, analyze[1].AgentEnabled.Value)
		require.NotNil(t, analyze[1].Reason)
		assert.Equal(t, string(odigosv1.AgentEnabledReasonUnsupportedProgrammingLanguage), analyze[1].Reason.Value)
		require.NotNil(t, analyze[1].Message)
		assert.Equal(t, "language not supported", analyze[1].Message.Value)
		assert.Nil(t, analyze[1].OtelDistroName)
	})

	t.Run("no containers", func(t *testing.T) {
		containers := []odigosv1.ContainerAgentConfig{}

		analyze := analyzeContainersConfig(&containers)

		assert.Empty(t, analyze)
	})
}

func TestAnalyzeRuntimeDetails(t *testing.T) {
	t.Run("detected runtime", func(t *testing.T) {
		criError := "failed to get env vars from CRI"
		details := []odigosv1.RuntimeDetailsByContainer{
			{
				ContainerName:  "checkout",
				Language:       common.GoProgrammingLanguage,
				RuntimeVersion: "1.24.0",
				EnvVars: []odigosv1.EnvVar{
					{Name: "GODEBUG", Value: "http2debug=1"},
				},
				EnvFromContainerRuntime: []odigosv1.EnvVar{
					{Name: "PYTHONPATH", Value: "/from/cri"},
				},
				CriErrorMessage: &criError,
			},
		}

		analyze := analyzeRuntimeDetails(details)

		require.Len(t, analyze, 1)
		container := analyze[0]
		assert.Equal(t, "checkout", container.ContainerName.Value)
		assert.True(t, container.ContainerName.ListKey)
		assert.Equal(t, common.GoProgrammingLanguage, container.Language.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, container.Language.Status)
		assert.Equal(t, "1.24.0", container.RuntimeVersion.Value)
		assert.Equal(t, criError, container.CriError.Value)
		assert.Equal(t, properties.PropertyStatusError, container.CriError.Status)
		require.Len(t, container.EnvVars, 1)
		assert.Equal(t, "GODEBUG", container.EnvVars[0].Name)
		assert.Equal(t, "http2debug=1", container.EnvVars[0].Value)
		require.Len(t, container.ContainerRuntimeEnvs, 1)
		assert.Equal(t, "PYTHONPATH", container.ContainerRuntimeEnvs[0].Name)
		assert.Equal(t, "/from/cri", container.ContainerRuntimeEnvs[0].Value)
	})

	t.Run("undetected runtime", func(t *testing.T) {
		details := []odigosv1.RuntimeDetailsByContainer{
			{ContainerName: "checkout", Language: common.UnknownProgrammingLanguage},
		}

		analyze := analyzeRuntimeDetails(details)

		require.Len(t, analyze, 1)
		assert.Equal(t, properties.PropertyStatusError, analyze[0].Language.Status)
		assert.Equal(t, "not available", analyze[0].RuntimeVersion.Value)
		assert.Equal(t, "No CRI error observed", analyze[0].CriError.Value)
		assert.Equal(t, properties.PropertyStatus(""), analyze[0].CriError.Status)
		assert.Empty(t, analyze[0].EnvVars)
		assert.Empty(t, analyze[0].ContainerRuntimeEnvs)
	})

	t.Run("every container keeps its own runtime details", func(t *testing.T) {
		details := []odigosv1.RuntimeDetailsByContainer{
			{ContainerName: "checkout", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.24.0"},
			{ContainerName: "frontend", Language: common.JavascriptProgrammingLanguage, RuntimeVersion: "20.11.0"},
		}

		analyze := analyzeRuntimeDetails(details)

		require.Len(t, analyze, 2)
		assert.Equal(t, "checkout", analyze[0].ContainerName.Value)
		assert.Equal(t, common.GoProgrammingLanguage, analyze[0].Language.Value)
		assert.Equal(t, "1.24.0", analyze[0].RuntimeVersion.Value)
		assert.Equal(t, "frontend", analyze[1].ContainerName.Value)
		assert.Equal(t, common.JavascriptProgrammingLanguage, analyze[1].Language.Value)
		assert.Equal(t, "20.11.0", analyze[1].RuntimeVersion.Value)
	})

	t.Run("no containers", func(t *testing.T) {
		assert.Empty(t, analyzeRuntimeDetails(nil))
	})
}

func TestAnalyzeRuntimeInfo(t *testing.T) {
	assert.Nil(t, analyzeRuntimeInfo(&OdigosSourceResources{}))

	resources := &OdigosSourceResources{
		InstrumentationConfig: &odigosv1.InstrumentationConfig{
			Status: odigosv1.InstrumentationConfigStatus{
				RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
					{ContainerName: "checkout", Language: common.GoProgrammingLanguage},
				},
			},
		},
	}

	runtimeInfo := analyzeRuntimeInfo(resources)

	require.NotNil(t, runtimeInfo)
	require.Len(t, runtimeInfo.Containers, 1)
	assert.Equal(t, "checkout", runtimeInfo.Containers[0].ContainerName.Value)
}

func TestAnalyzeInstrumentationInstance(t *testing.T) {
	t.Run("health not reported yet", func(t *testing.T) {
		analyze := analyzeInstrumentationInstance(&odigosv1.InstrumentationInstance{})

		assert.Equal(t, "Not Reported", analyze.Healthy.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, analyze.Healthy.Status)
		assert.True(t, analyze.Healthy.ListKey)
		assert.Nil(t, analyze.Message)
		assert.Empty(t, analyze.IdentifyingAttributes)
	})

	t.Run("healthy instrumentation", func(t *testing.T) {
		healthy := true
		instance := &odigosv1.InstrumentationInstance{
			Status: odigosv1.InstrumentationInstanceStatus{
				Healthy: &healthy,
				IdentifyingAttributes: []odigosv1.Attribute{
					{Key: "service.name", Value: "checkout"},
					{Key: "telemetry.sdk.language", Value: "go"},
				},
			},
		}

		analyze := analyzeInstrumentationInstance(instance)

		assert.Equal(t, true, analyze.Healthy.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.Healthy.Status)
		assert.Nil(t, analyze.Message)
		require.Len(t, analyze.IdentifyingAttributes, 2)
		assert.Equal(t, "service.name", analyze.IdentifyingAttributes[0].Name)
		assert.Equal(t, "checkout", analyze.IdentifyingAttributes[0].Value)
		assert.Equal(t, "telemetry.sdk.language", analyze.IdentifyingAttributes[1].Name)
		assert.Equal(t, "go", analyze.IdentifyingAttributes[1].Value)
	})

	t.Run("unhealthy instrumentation reports its message", func(t *testing.T) {
		healthy := false
		instance := &odigosv1.InstrumentationInstance{
			Status: odigosv1.InstrumentationInstanceStatus{
				Healthy: &healthy,
				Message: "failed to load eBPF probes",
			},
		}

		analyze := analyzeInstrumentationInstance(instance)

		assert.Equal(t, false, analyze.Healthy.Value)
		assert.Equal(t, properties.PropertyStatusError, analyze.Healthy.Status)
		require.NotNil(t, analyze.Message)
		assert.Equal(t, "failed to load eBPF probes", analyze.Message.Value)
	})
}

func TestPodPhaseToStatus(t *testing.T) {
	testCases := map[corev1.PodPhase]properties.PropertyStatus{
		corev1.PodRunning:   properties.PropertyStatusSuccess,
		corev1.PodSucceeded: properties.PropertyStatusSuccess,
		corev1.PodPending:   properties.PropertyStatusTransitioning,
		corev1.PodFailed:    properties.PropertyStatusError,
		corev1.PodUnknown:   properties.PropertyStatusError,
		corev1.PodPhase(""): properties.PropertyStatusError,
	}

	for phase, expected := range testCases {
		t.Run(string(phase), func(t *testing.T) {
			assert.Equal(t, expected, podPhaseToStatus(phase))
		})
	}
}

func TestGetContainerStatus(t *testing.T) {
	pod := &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
		{Name: "istio-proxy", Ready: false},
		{Name: "checkout", Ready: true},
	}}}

	status := getContainerStatus(pod, "checkout")
	require.NotNil(t, status)
	assert.Equal(t, "checkout", status.Name)
	assert.True(t, status.Ready)
	// the returned status must point into the live pod object
	assert.Same(t, &pod.Status.ContainerStatuses[1], status)

	assert.Nil(t, getContainerStatus(pod, "not-in-the-pod"))
	assert.Nil(t, getContainerStatus(&corev1.Pod{}, "checkout"))
}

func TestAnalyzePodContainer(t *testing.T) {
	t.Run("container without a status has no started or ready properties", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		resources := &OdigosSourceResources{
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		}

		analyze := analyzePodContainer(pod, &pod.Spec.Containers[0], resources)

		assert.Equal(t, "checkout", analyze.ContainerName.Value)
		assert.True(t, analyze.ContainerName.ListKey)
		assert.Empty(t, analyze.ActualDevices.Value)
		assert.Equal(t, properties.EntityProperty{}, analyze.Started)
		assert.Equal(t, properties.EntityProperty{}, analyze.Ready)
		assert.Empty(t, analyze.InstrumentationInstances)
	})

	t.Run("started and ready are read from the container status", func(t *testing.T) {
		started := true
		pod := analyzeTestPod("checkout-1")
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{Name: "checkout", Started: &started, Ready: false},
		}
		resources := &OdigosSourceResources{
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		}

		analyze := analyzePodContainer(pod, &pod.Spec.Containers[0], resources)

		assert.Equal(t, true, analyze.Started.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.Started.Status)
		assert.Equal(t, false, analyze.Ready.Value)
		assert.Equal(t, properties.PropertyStatusError, analyze.Ready.Status)
	})

	t.Run("an explicitly false started flag is reported as not started", func(t *testing.T) {
		started := false
		pod := analyzeTestPod("checkout-1")
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{Name: "checkout", Started: &started, Ready: false},
		}
		resources := &OdigosSourceResources{
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		}

		analyze := analyzePodContainer(pod, &pod.Spec.Containers[0], resources)

		assert.Equal(t, false, analyze.Started.Value)
		assert.Equal(t, properties.PropertyStatusError, analyze.Started.Status)
	})

	t.Run("a nil started flag is reported as not started", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{Name: "checkout", Started: nil, Ready: true},
		}
		resources := &OdigosSourceResources{
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		}

		analyze := analyzePodContainer(pod, &pod.Spec.Containers[0], resources)

		assert.Equal(t, false, analyze.Started.Value)
		assert.Equal(t, properties.PropertyStatusError, analyze.Started.Status)
		assert.Equal(t, true, analyze.Ready.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.Ready.Status)
	})

	t.Run("only odigos instrumentation devices are reported", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceName(common.OdigosResourceNamespace + "/golang-community"): resource.MustParse("1"),
				corev1.ResourceCPU: resource.MustParse("1"),
				"nvidia.com/gpu":   resource.MustParse("1"),
			},
		}
		resources := &OdigosSourceResources{
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		}

		analyze := analyzePodContainer(pod, &pod.Spec.Containers[0], resources)

		assert.Equal(t, []string{"golang-community"}, analyze.ActualDevices.Value)
	})

	t.Run("only the instrumentation instances of this pod and container are reported", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		otherContainerInstance := ownedInstrumentationInstance("checkout-1", "sidecar")
		otherPodInstance := ownedInstrumentationInstance("checkout-2", "checkout")

		noOwnerInstance := ownedInstrumentationInstance("checkout-1", "checkout")
		noOwnerInstance.Name = "no-owner"
		noOwnerInstance.OwnerReferences = nil

		twoOwnersInstance := ownedInstrumentationInstance("checkout-1", "checkout")
		twoOwnersInstance.Name = "two-owners"
		twoOwnersInstance.OwnerReferences = append(twoOwnersInstance.OwnerReferences,
			metav1.OwnerReference{Kind: "ReplicaSet", Name: "checkout-abc"})

		notOwnedByPodInstance := ownedInstrumentationInstance("checkout-1", "checkout")
		notOwnedByPodInstance.Name = "owned-by-replicaset"
		notOwnedByPodInstance.OwnerReferences = []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "checkout-1"}}

		healthy := true
		matchingInstance := ownedInstrumentationInstance("checkout-1", "checkout")
		matchingInstance.Status.Healthy = &healthy

		resources := &OdigosSourceResources{
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{Items: []odigosv1.InstrumentationInstance{
				otherContainerInstance,
				otherPodInstance,
				noOwnerInstance,
				twoOwnersInstance,
				notOwnedByPodInstance,
				matchingInstance,
			}},
		}

		analyze := analyzePodContainer(pod, &pod.Spec.Containers[0], resources)

		require.Len(t, analyze.InstrumentationInstances, 1)
		assert.Equal(t, true, analyze.InstrumentationInstances[0].Healthy.Value)
	})
}

func TestAnalyzePod(t *testing.T) {
	resources := func() *OdigosSourceResources {
		return &OdigosSourceResources{InstrumentationInstances: &odigosv1.InstrumentationInstanceList{}}
	}

	t.Run("pod without the odigos agents hash label", func(t *testing.T) {
		analyze := analyzePod(analyzeTestPod("checkout-1"), resources())

		assert.Equal(t, "checkout-1", analyze.PodName.Value)
		assert.True(t, analyze.PodName.ListKey)
		assert.Equal(t, "node-1", analyze.NodeName.Value)
		assert.Equal(t, false, analyze.AgentInjected.Value)
		assert.Equal(t, properties.PropertyStatusError, analyze.AgentInjected.Status)
		assert.Nil(t, analyze.RunningLatestWorkloadRevision)
		assert.Equal(t, corev1.PodRunning, analyze.Phase.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.Phase.Status)
		require.Len(t, analyze.Containers, 1)
		assert.Equal(t, "checkout", analyze.Containers[0].ContainerName.Value)
	})

	t.Run("pod with the odigos agents hash label", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Labels = map[string]string{k8sconsts.OdigosAgentsMetaHashLabel: "abc123"}

		analyze := analyzePod(pod, resources())

		assert.Equal(t, true, analyze.AgentInjected.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.AgentInjected.Status)
	})

	// an empty hash label value still means the webhook injected the agent
	t.Run("an empty agents hash label value still counts as injected", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Labels = map[string]string{k8sconsts.OdigosAgentsMetaHashLabel: ""}

		analyze := analyzePod(pod, resources())

		assert.Equal(t, true, analyze.AgentInjected.Value)
	})

	t.Run("running the latest workload revision", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Annotations = map[string]string{OdigosRunningLatestWorkloadRevisionAnnotation: "true"}

		analyze := analyzePod(pod, resources())

		require.NotNil(t, analyze.RunningLatestWorkloadRevision)
		assert.Equal(t, true, analyze.RunningLatestWorkloadRevision.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.RunningLatestWorkloadRevision.Status)
	})

	t.Run("running an older workload revision", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Annotations = map[string]string{OdigosRunningLatestWorkloadRevisionAnnotation: "false"}

		analyze := analyzePod(pod, resources())

		require.NotNil(t, analyze.RunningLatestWorkloadRevision)
		assert.Equal(t, false, analyze.RunningLatestWorkloadRevision.Value)
		assert.Equal(t, properties.PropertyStatusError, analyze.RunningLatestWorkloadRevision.Status)
	})

	t.Run("a terminating pod is reported as transitioning", func(t *testing.T) {
		deletionTime := metav1.Now()
		pod := analyzeTestPod("checkout-1")
		pod.DeletionTimestamp = &deletionTime

		analyze := analyzePod(pod, resources())

		assert.Equal(t, "Terminating", analyze.Phase.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, analyze.Phase.Status)
	})

	t.Run("a failed pod is reported as an error", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Status.Phase = corev1.PodFailed

		analyze := analyzePod(pod, resources())

		assert.Equal(t, corev1.PodFailed, analyze.Phase.Value)
		assert.Equal(t, properties.PropertyStatusError, analyze.Phase.Status)
	})

	t.Run("every container of the pod is analyzed", func(t *testing.T) {
		pod := analyzeTestPod("checkout-1")
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "istio-proxy"})

		analyze := analyzePod(pod, resources())

		require.Len(t, analyze.Containers, 2)
		assert.Equal(t, "checkout", analyze.Containers[0].ContainerName.Value)
		assert.Equal(t, "istio-proxy", analyze.Containers[1].ContainerName.Value)
	})
}

func TestAnalyzePods(t *testing.T) {
	t.Run("no pods", func(t *testing.T) {
		pods, phasesText := analyzePods(&OdigosSourceResources{
			Pods:                     &corev1.PodList{},
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		})

		assert.Empty(t, pods)
		assert.Equal(t, "", phasesText)
	})

	t.Run("pods of a single phase", func(t *testing.T) {
		running := analyzeTestPod("checkout-1")
		alsoRunning := analyzeTestPod("checkout-2")

		pods, phasesText := analyzePods(&OdigosSourceResources{
			Pods:                     &corev1.PodList{Items: []corev1.Pod{*running, *alsoRunning}},
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		})

		require.Len(t, pods, 2)
		assert.Equal(t, "checkout-1", pods[0].PodName.Value)
		assert.Equal(t, "checkout-2", pods[1].PodName.Value)
		assert.Equal(t, "Running 2", phasesText)
	})

	t.Run("pods of several phases are counted per phase", func(t *testing.T) {
		running := analyzeTestPod("checkout-1")
		pending := analyzeTestPod("checkout-2")
		pending.Status.Phase = corev1.PodPending
		alsoPending := analyzeTestPod("checkout-3")
		alsoPending.Status.Phase = corev1.PodPending

		_, phasesText := analyzePods(&OdigosSourceResources{
			Pods:                     &corev1.PodList{Items: []corev1.Pod{*running, *pending, *alsoPending}},
			InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		})

		// the phases are joined from a map, so only the set of counts is deterministic
		assert.ElementsMatch(t, []string{"Running 1", "Pending 2"}, strings.Split(phasesText, ", "))
	})
}

func TestAnalyzeSource(t *testing.T) {
	workloadObj := &K8sSourceObject{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "shop"},
		Kind:       k8sconsts.WorkloadKindDeployment,
	}
	healthy := true
	resources := &OdigosSourceResources{
		Sources: &odigosv1.WorkloadSources{Workload: sourceObject("workload-source", false)},
		InstrumentationConfig: &odigosv1.InstrumentationConfig{
			Spec: odigosv1.InstrumentationConfigSpec{
				Containers: []odigosv1.ContainerAgentConfig{
					{ContainerName: "checkout", AgentEnabled: true, OtelDistroName: "golang-community"},
				},
			},
			Status: odigosv1.InstrumentationConfigStatus{
				RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
					{ContainerName: "checkout", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.24.0"},
				},
			},
		},
		InstrumentationInstances: &odigosv1.InstrumentationInstanceList{Items: []odigosv1.InstrumentationInstance{
			func() odigosv1.InstrumentationInstance {
				instance := ownedInstrumentationInstance("checkout-1", "checkout")
				instance.Status.Healthy = &healthy
				return instance
			}(),
		}},
		Pods: &corev1.PodList{Items: []corev1.Pod{*analyzeTestPod("checkout-1")}},
	}

	analyze := AnalyzeSource(resources, workloadObj)

	assert.Equal(t, "checkout", analyze.Name.Value)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, analyze.Kind.Value)
	assert.Equal(t, "shop", analyze.Namespace.Value)
	assert.Equal(t, true, analyze.SourceObjectsAnalysis.Instrumented.Value)
	require.NotNil(t, analyze.RuntimeInfo)
	require.Len(t, analyze.RuntimeInfo.Containers, 1)
	assert.Equal(t, common.GoProgrammingLanguage, analyze.RuntimeInfo.Containers[0].Language.Value)
	assert.Equal(t, "created", analyze.OtelAgents.Created.Value)
	assert.Equal(t, properties.PropertyStatusSuccess, analyze.OtelAgents.Created.Status)
	require.Len(t, analyze.OtelAgents.Containers, 1)
	assert.Equal(t, 1, analyze.TotalPods)
	assert.Equal(t, "Running 1", analyze.PodsPhasesCount)
	require.Len(t, analyze.Pods, 1)
	require.Len(t, analyze.Pods[0].Containers, 1)
	require.Len(t, analyze.Pods[0].Containers[0].InstrumentationInstances, 1)
	assert.Equal(t, true, analyze.Pods[0].Containers[0].InstrumentationInstances[0].Healthy.Value)
}

// an uninstrumented workload with a leftover instrumentation config is the shape that
// tells a user their Source was removed but odigos has not finished cleaning up
func TestAnalyzeSourceUninstrumentedWorkload(t *testing.T) {
	workloadObj := &K8sSourceObject{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "shop"},
		Kind:       k8sconsts.WorkloadKindStatefulSet,
	}
	resources := &OdigosSourceResources{
		Sources:                  &odigosv1.WorkloadSources{},
		InstrumentationInstances: &odigosv1.InstrumentationInstanceList{},
		Pods:                     &corev1.PodList{},
	}

	analyze := AnalyzeSource(resources, workloadObj)

	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, analyze.Kind.Value)
	assert.Equal(t, false, analyze.SourceObjectsAnalysis.Instrumented.Value)
	assert.Nil(t, analyze.RuntimeInfo)
	assert.Equal(t, "not created", analyze.OtelAgents.Created.Value)
	assert.Equal(t, properties.PropertyStatusSuccess, analyze.OtelAgents.Created.Status)
	assert.Empty(t, analyze.OtelAgents.Containers)
	assert.Equal(t, 0, analyze.TotalPods)
	assert.Empty(t, analyze.Pods)
}
