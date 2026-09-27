package agentenabled

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// ****************
// Webhook test fixtures
// ****************

const (
	webhookNamespace   = "default"
	webhookWorkload    = "my-deployment"
	webhookContainer   = "my-container"
	webhookServiceName = "my-service"
	webhookAgentsHash  = "agents-meta-hash"
	// nodejs-community mounts two agent directories, declares a virtual device, sets k8s
	// attributes via env vars and supports ld-preload injection, which exercises every
	// branch of the per container injection.
	webhookDistroName = "nodejs-community"
)

func webhookTestContext() context.Context {
	return logr.NewContext(context.Background(), logr.Discard())
}

func newWebhookTestConfig(mountMethod common.MountMethod) common.OdigosConfiguration {
	envInjectionMethod := common.PodManifestEnvInjectionMethod
	return common.OdigosConfiguration{
		MountMethod:                 &mountMethod,
		AgentEnvVarsInjectionMethod: &envInjectionMethod,
		AgentsInitContainerResources: &common.AgentsInitContainerResources{
			RequestCPUm:      100,
			LimitCPUm:        200,
			RequestMemoryMiB: 300,
			LimitMemoryMiB:   400,
		},
	}
}

// newWebhookTestPodsWebhook builds a PodsWebhook backed by a fake client that holds the given
// effective odigos configuration (as the scheduler would write it) and the given objects.
func newWebhookTestPodsWebhook(t *testing.T, config common.OdigosConfiguration, objects ...client.Object) *PodsWebhook {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, admissionv1.AddToScheme(scheme))

	// the effective config is read with sigs.k8s.io/yaml, which accepts json
	configJSON, err := json.Marshal(config)
	require.NoError(t, err)
	objects = append(objects, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      commonconsts.OdigosEffectiveConfigName,
			Namespace: commonconsts.DefaultOdigosNamespace,
		},
		Data: map[string]string{commonconsts.OdigosConfigurationFileName: string(configJSON)},
	})

	getter, err := distros.NewCommunityGetter()
	require.NoError(t, err)

	return &PodsWebhook{
		Client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(),
		DistrosGetter: getter,
		Decoder:       admission.NewDecoder(scheme),
	}
}

func newWebhookTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookWorkload + "-abc123-xyz",
			Namespace: webhookNamespace,
			Labels:    map[string]string{"app": webhookWorkload},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       webhookWorkload + "-abc123",
				UID:        "rs-uid",
			}},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: webhookContainer, Image: "my-image"}},
		},
	}
}

func newWebhookTestIC() *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName(webhookWorkload, k8sconsts.WorkloadKindDeployment),
			Namespace: webhookNamespace,
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			ServiceName:           webhookServiceName,
			AgentInjectionEnabled: true,
			AgentsMetaHash:        webhookAgentsHash,
			Containers: []odigosv1.ContainerAgentConfig{{
				ContainerName:  webhookContainer,
				AgentEnabled:   true,
				OtelDistroName: webhookDistroName,
			}},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{{
				ContainerName: webhookContainer,
				Language:      common.JavascriptProgrammingLanguage,
			}},
		},
	}
}

func webhookPodEnvValue(t *testing.T, pod *corev1.Pod, containerIndex int, envName string) string {
	t.Helper()
	for _, envVar := range pod.Spec.Containers[containerIndex].Env {
		if envVar.Name == envName {
			return envVar.Value
		}
	}
	require.Failf(t, "env var not found", "env var %s is missing from container %s", envName,
		pod.Spec.Containers[containerIndex].Name)
	return ""
}

func webhookPodHasEnv(pod *corev1.Pod, containerIndex int, envName string) bool {
	for _, envVar := range pod.Spec.Containers[containerIndex].Env {
		if envVar.Name == envName {
			return true
		}
	}
	return false
}

func webhookOdigletAffinityKeys(pod *corev1.Pod) []string {
	keys := []string{}
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.NodeAffinity == nil ||
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return keys
	}
	for _, term := range pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			keys = append(keys, expr.Key)
		}
	}
	return keys
}

// ****************
// Handle: user workloads must never be blocked by instrumentation logic
// ****************

func TestPodsWebhookHandle(t *testing.T) {
	newRequest := func(t *testing.T, pod *corev1.Pod) admission.Request {
		t.Helper()
		raw, err := json.Marshal(pod)
		require.NoError(t, err)
		return admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: webhookNamespace,
			Object:    runtime.RawExtension{Raw: raw},
		}}
	}

	t.Run("mutates an instrumented pod", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())

		resp := p.Handle(webhookTestContext(), newRequest(t, newWebhookTestPod()))

		assert.True(t, resp.Allowed)
		require.NotEmpty(t, resp.Patches)
		patchPaths := make([]string, 0, len(resp.Patches))
		for _, patch := range resp.Patches {
			patchPaths = append(patchPaths, patch.Path)
		}
		assert.Contains(t, patchPaths, "/metadata/labels/odigos.io~1agents-meta-hash")
		assert.Contains(t, patchPaths, "/spec/containers/0/env")
	})

	t.Run("an undecodable object is allowed without mutation", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())

		resp := p.Handle(webhookTestContext(), admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: webhookNamespace,
			Object:    runtime.RawExtension{Raw: []byte("not-a-pod")},
		}})

		assert.True(t, resp.Allowed)
		assert.Empty(t, resp.Patches)
		require.NotNil(t, resp.Result)
		assert.Equal(t, "proceeding without mutation", resp.Result.Message)
	})

	t.Run("a pod without a workload owner is skipped", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()
		pod.OwnerReferences = nil

		resp := p.Handle(webhookTestContext(), newRequest(t, pod))

		assert.True(t, resp.Allowed)
		assert.Empty(t, resp.Patches)
		require.NotNil(t, resp.Result)
		assert.Equal(t, "odigos injection skipped: not applicable", resp.Result.Message)
	})

	t.Run("a pod of a workload odigos does not instrument is skipped", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod))

		resp := p.Handle(webhookTestContext(), newRequest(t, newWebhookTestPod()))

		assert.True(t, resp.Allowed)
		assert.Empty(t, resp.Patches)
		require.NotNil(t, resp.Result)
		assert.Equal(t, "odigos injection skipped: not applicable", resp.Result.Message)
	})

	t.Run("agent injection disabled is skipped", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.AgentInjectionEnabled = false
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)

		resp := p.Handle(webhookTestContext(), newRequest(t, newWebhookTestPod()))

		assert.True(t, resp.Allowed)
		assert.Empty(t, resp.Patches)
		require.NotNil(t, resp.Result)
		assert.Equal(t, "odigos injection skipped: not applicable", resp.Result.Message)
	})

	t.Run("an internal failure still allows the pod", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		// remove the effective config the injection depends on
		require.NoError(t, p.Delete(context.Background(), &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonconsts.OdigosEffectiveConfigName,
				Namespace: commonconsts.DefaultOdigosNamespace,
			},
		}))

		resp := p.Handle(webhookTestContext(), newRequest(t, newWebhookTestPod()))

		assert.True(t, resp.Allowed)
		assert.Empty(t, resp.Patches)
		require.NotNil(t, resp.Result)
		assert.Equal(t, "odigos injection failed internally", resp.Result.Message)
	})
}

// ****************
// injectOdigos: mount methods
// ****************

func TestInjectOdigosMountMethods(t *testing.T) {
	ctx := webhookTestContext()
	req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{Namespace: webhookNamespace}}

	t.Run("virtual device", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Equal(t, resource.MustParse("1"),
			pod.Spec.Containers[0].Resources.Limits["instrumentation.odigos.io/generic"])
		assert.Empty(t, pod.Spec.Volumes)
		assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)
		assert.Empty(t, pod.Spec.InitContainers)
		// the device guarantees odiglet is on the node, so no node affinity is needed
		assert.Empty(t, webhookOdigletAffinityKeys(pod))
	})

	t.Run("host path", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sHostPathMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		require.Len(t, pod.Spec.Volumes, 1)
		require.NotNil(t, pod.Spec.Volumes[0].HostPath)
		assert.Equal(t, "/var/odigos", pod.Spec.Volumes[0].HostPath.Path)
		mountPaths := []string{}
		for _, mount := range pod.Spec.Containers[0].VolumeMounts {
			mountPaths = append(mountPaths, mount.MountPath)
		}
		assert.ElementsMatch(t, []string{"/var/odigos/nodejs-community", "/var/odigos/opentelemetry-node"}, mountPaths)
		// no device is requested when the agent is not mounted via the virtual device
		assert.Empty(t, pod.Spec.Containers[0].Resources.Limits)
		assert.Equal(t, []string{"odigos.io/odiglet-oss-installed"}, webhookOdigletAffinityKeys(pod))
	})

	t.Run("csi driver", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sCsiDriverMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		require.Len(t, pod.Spec.Volumes, 1)
		require.NotNil(t, pod.Spec.Volumes[0].CSI)
		assert.Equal(t, "odigos.csi.driver", pod.Spec.Volumes[0].CSI.Driver)
		assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 2)
		assert.Equal(t, []string{"odigos.io/odiglet-oss-installed"}, webhookOdigletAffinityKeys(pod))
	})

	t.Run("init container", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sInitContainerMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		require.Len(t, pod.Spec.Volumes, 1)
		require.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
		require.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, []string{
			"sh", "-c",
			"cp -r /instrumentations/nodejs-community /var/odigos/nodejs-community && " +
				"cp -r /instrumentations/opentelemetry-node /var/odigos/opentelemetry-node",
		}, pod.Spec.InitContainers[0].Command)
		// the init container writes the agent files, odiglet does not have to prepare the node
		assert.Empty(t, webhookOdigletAffinityKeys(pod))
	})

	t.Run("karpenter installations do not get the odiglet node affinity", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sHostPathMountMethod)
		karpenterEnabled := true
		config.KarpenterEnabled = &karpenterEnabled
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Empty(t, webhookOdigletAffinityKeys(pod))
		// the agent is still mounted
		assert.Len(t, pod.Spec.Volumes, 1)
	})

	t.Run("loader injection also mounts and copies the loader directory", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sInitContainerMountMethod)
		loaderInjection := common.LoaderFallbackToPodManifestInjectionMethod
		config.AgentEnvVarsInjectionMethod = &loaderInjection
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		mountPaths := []string{}
		for _, mount := range pod.Spec.Containers[0].VolumeMounts {
			mountPaths = append(mountPaths, mount.MountPath)
		}
		assert.Contains(t, mountPaths, "/var/odigos/loader")
		require.Len(t, pod.Spec.InitContainers, 1)
		assert.Contains(t, pod.Spec.InitContainers[0].Command[2], "cp -r /instrumentations/loader /var/odigos/loader")
	})

	t.Run("a distro without a runtime agent directory does not mount a volume", func(t *testing.T) {
		ic := newWebhookTestIC()
		// golang-community is an ebpf distro: it has no agent directories to mount
		ic.Spec.Containers[0].OtelDistroName = "golang-community"
		ic.Status.RuntimeDetailsByContainer[0].Language = common.GoProgrammingLanguage
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sHostPathMountMethod), ic)
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Empty(t, pod.Spec.Volumes)
		assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)
	})
}

// ****************
// injectOdigos: per container mutations
// ****************

func TestInjectOdigosContainers(t *testing.T) {
	ctx := webhookTestContext()
	req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{Namespace: webhookNamespace}}

	t.Run("injects the odigos env vars and the agents meta hash label", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Equal(t, webhookContainer, webhookPodEnvValue(t, pod, 0, "ODIGOS_CONTAINER_NAME"))
		assert.Equal(t, webhookDistroName, webhookPodEnvValue(t, pod, 0, "ODIGOS_DISTRO_NAME"))
		assert.Equal(t, webhookNamespace, webhookPodEnvValue(t, pod, 0, "ODIGOS_WORKLOAD_NAMESPACE"))
		assert.Equal(t, webhookServiceName, webhookPodEnvValue(t, pod, 0, "OTEL_SERVICE_NAME"))
		assert.Contains(t, webhookPodEnvValue(t, pod, 0, "OTEL_RESOURCE_ATTRIBUTES"),
			"k8s.deployment.name="+webhookWorkload)
		// the pod is associated with the instrumentation version it was created with
		assert.Equal(t, webhookAgentsHash, pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel])
	})

	t.Run("injects the agent into the container manifest", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Equal(t, "--require /var/odigos/nodejs-community/autoinstrumentation.js",
			webhookPodEnvValue(t, pod, 0, "NODE_OPTIONS"))
		// this distro reports its enabled signals dynamically, so the static otel exporter
		// env vars (which are frozen at pod creation time) must not be set
		assert.False(t, webhookPodHasEnv(pod, 0, "OTEL_TRACES_EXPORTER"))
		assert.False(t, webhookPodHasEnv(pod, 0, "OTEL_METRICS_EXPORTER"))
		assert.False(t, webhookPodHasEnv(pod, 0, "OTEL_LOGS_EXPORTER"))
	})

	t.Run("a container without an agent config is not touched", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "sidecar", Image: "sidecar-image"})

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.NotEmpty(t, pod.Spec.Containers[0].Env)
		assert.Empty(t, pod.Spec.Containers[1].Env)
		assert.Empty(t, pod.Spec.Containers[1].Resources.Limits)
	})

	t.Run("a container with the agent disabled is not touched", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.Containers[0].AgentEnabled = false
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Empty(t, pod.Spec.Containers[0].Env)
		assert.Empty(t, pod.Spec.Containers[0].Resources.Limits)
		// the label is still written, so the pod is tracked as up to date with the config
		assert.Equal(t, webhookAgentsHash, pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel])
	})

	t.Run("a container with an empty distro name is not touched", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.Containers[0].OtelDistroName = ""
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Empty(t, pod.Spec.Containers[0].Env)
	})

	t.Run("a container with unknown language is not env injected", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Status.RuntimeDetailsByContainer[0].Language = common.UnknownProgrammingLanguage
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		// the device and the odigos env vars are still injected
		assert.Equal(t, resource.MustParse("1"),
			pod.Spec.Containers[0].Resources.Limits["instrumentation.odigos.io/generic"])
		assert.False(t, webhookPodHasEnv(pod, 0, "NODE_OPTIONS"))
	})

	t.Run("user instrumentation env vars are injected when configured", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sVirtualDeviceMountMethod)
		config.UserInstrumentationEnvs = &common.UserInstrumentationEnvs{
			Languages: map[common.ProgrammingLanguage]common.LanguageConfig{
				common.JavascriptProgrammingLanguage: {Enabled: true, EnvVars: map[string]string{"MY_USER_VAR": "my-value"}},
			},
		}
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Equal(t, "my-value", webhookPodEnvValue(t, pod, 0, "MY_USER_VAR"))
	})

	t.Run("the wasp mutator is not called for a distro that does not support wasp", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sVirtualDeviceMountMethod)
		waspEnabled := true
		config.WaspEnabled = &waspEnabled
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())
		waspCalled := false
		p.WaspMutator = func(*corev1.Pod, common.OdigosConfiguration) error {
			waspCalled = true
			return nil
		}

		require.NoError(t, p.injectOdigos(ctx, newWebhookTestPod(), req))

		assert.False(t, waspCalled)
	})
}

// ****************
// injectOdigos: failures
// ****************

func TestInjectOdigosErrors(t *testing.T) {
	ctx := webhookTestContext()
	req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{Namespace: webhookNamespace}}

	t.Run("mount method not resolved in the effective config", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sVirtualDeviceMountMethod)
		config.MountMethod = nil
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())

		err := p.injectOdigos(ctx, newWebhookTestPod(), req)

		assert.ErrorIs(t, err, ErrMountMethodNotSet)
	})

	t.Run("missing service name", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.ServiceName = ""
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)

		err := p.injectOdigos(ctx, newWebhookTestPod(), req)

		assert.ErrorIs(t, err, ErrMissingServiceName)
	})

	t.Run("unknown distro name", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.Containers[0].OtelDistroName = "no-such-distro"
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)

		err := p.injectOdigos(ctx, newWebhookTestPod(), req)

		assert.ErrorIs(t, err, ErrUnknownDistroName)
	})

	t.Run("effective config missing", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		require.NoError(t, p.Delete(context.Background(), &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonconsts.OdigosEffectiveConfigName,
				Namespace: commonconsts.DefaultOdigosNamespace,
			},
		}))

		err := p.injectOdigos(ctx, newWebhookTestPod(), req)

		assert.ErrorIs(t, err, ErrMissingOdigosConfiguration)
	})

	t.Run("env injection method not resolved in the effective config", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sVirtualDeviceMountMethod)
		config.AgentEnvVarsInjectionMethod = nil
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())

		err := p.injectOdigos(ctx, newWebhookTestPod(), req)

		assert.ErrorIs(t, err, ErrEnvVarInjection)
	})

	t.Run("device health check blocks injection when the device plugin is unhealthy", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sVirtualDeviceMountMethod)
		checkDeviceHealth := true
		config.CheckDeviceHealthBeforeInjection = &checkDeviceHealth
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())

		// no odiglet daemonset exists in the odigos namespace
		err := p.injectOdigos(ctx, newWebhookTestPod(), req)

		assert.ErrorIs(t, err, ErrOdigletDeviceNotHealthy)
	})

	t.Run("device health check is skipped for mount methods that do not use the device", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sHostPathMountMethod)
		checkDeviceHealth := true
		config.CheckDeviceHealthBeforeInjection = &checkDeviceHealth
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())

		require.NoError(t, p.injectOdigos(ctx, newWebhookTestPod(), req))
	})

	t.Run("device health check is skipped when disabled", func(t *testing.T) {
		config := newWebhookTestConfig(common.K8sVirtualDeviceMountMethod)
		checkDeviceHealth := false
		config.CheckDeviceHealthBeforeInjection = &checkDeviceHealth
		p := newWebhookTestPodsWebhook(t, config, newWebhookTestIC())

		require.NoError(t, p.injectOdigos(ctx, newWebhookTestPod(), req))
	})

	t.Run("a pod in a namespace that only the admission request knows", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()
		// pods are not namespaced yet when the create request is admitted
		pod.Namespace = ""

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Equal(t, webhookNamespace, webhookPodEnvValue(t, pod, 0, "ODIGOS_WORKLOAD_NAMESPACE"))
	})

	t.Run("a pod with no namespace at all is ignored", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newWebhookTestIC())
		pod := newWebhookTestPod()
		pod.Namespace = ""

		err := p.injectOdigos(ctx, pod, admission.Request{})

		assert.ErrorIs(t, err, ErrIgnorePod)
	})
}

// ****************
// getRuntimeInfoForContainerName
// ****************

func TestGetRuntimeInfoForContainerName(t *testing.T) {
	runtimeVersion := "v18.2.0"

	t.Run("an override wins over the detected runtime info", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.ContainersOverrides = []odigosv1.ContainerOverride{{
			ContainerName: webhookContainer,
			RuntimeInfo: &odigosv1.RuntimeDetailsByContainer{
				ContainerName:  webhookContainer,
				Language:       common.PythonProgrammingLanguage,
				RuntimeVersion: runtimeVersion,
			},
		}}

		runtimeInfo := getRuntimeInfoForContainerName(ic, webhookContainer)

		require.NotNil(t, runtimeInfo)
		assert.Equal(t, common.PythonProgrammingLanguage, runtimeInfo.Language)
	})

	t.Run("an override without runtime info falls back to the detected runtime info", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.ContainersOverrides = []odigosv1.ContainerOverride{{ContainerName: webhookContainer}}

		runtimeInfo := getRuntimeInfoForContainerName(ic, webhookContainer)

		require.NotNil(t, runtimeInfo)
		assert.Equal(t, common.JavascriptProgrammingLanguage, runtimeInfo.Language)
	})

	t.Run("an override of another container is not used", func(t *testing.T) {
		ic := newWebhookTestIC()
		ic.Spec.ContainersOverrides = []odigosv1.ContainerOverride{{
			ContainerName: "other-container",
			RuntimeInfo: &odigosv1.RuntimeDetailsByContainer{
				ContainerName: "other-container",
				Language:      common.PythonProgrammingLanguage,
			},
		}}

		runtimeInfo := getRuntimeInfoForContainerName(ic, webhookContainer)

		require.NotNil(t, runtimeInfo)
		assert.Equal(t, common.JavascriptProgrammingLanguage, runtimeInfo.Language)
	})

	t.Run("no runtime info for the container", func(t *testing.T) {
		ic := newWebhookTestIC()

		assert.Nil(t, getRuntimeInfoForContainerName(ic, "other-container"))
	})
}

// ****************
// createInitContainer / getInitContainerImage
// ****************

func TestCreateInitContainer(t *testing.T) {
	config := newWebhookTestConfig(common.K8sInitContainerMountMethod)

	t.Run("copies every directory in a deterministic order", func(t *testing.T) {
		pod := newWebhookTestPod()
		dirsToCopy := map[string]struct{}{
			"{{ODIGOS_AGENTS_DIR}}/opentelemetry-node": {},
			"{{ODIGOS_AGENTS_DIR}}/loader":             {},
			"{{ODIGOS_AGENTS_DIR}}/nodejs-community":   {},
		}

		createInitContainer(pod, dirsToCopy, config)

		require.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "cp -r /instrumentations/loader /var/odigos/loader && "+
			"cp -r /instrumentations/nodejs-community /var/odigos/nodejs-community && "+
			"cp -r /instrumentations/opentelemetry-node /var/odigos/opentelemetry-node",
			pod.Spec.InitContainers[0].Command[2])
	})

	t.Run("mounts the shared agents volume and drops privileges", func(t *testing.T) {
		pod := newWebhookTestPod()

		createInitContainer(pod, map[string]struct{}{"{{ODIGOS_AGENTS_DIR}}/nodejs-community": {}}, config)

		require.Len(t, pod.Spec.InitContainers, 1)
		initContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, k8sconsts.OdigosInitContainerName, initContainer.Name)
		assert.Equal(t, corev1.PullIfNotPresent, initContainer.ImagePullPolicy)
		assert.Equal(t, []corev1.VolumeMount{{Name: "odigos-agent", MountPath: "/var/odigos"}}, initContainer.VolumeMounts)
		require.NotNil(t, initContainer.SecurityContext)
		require.NotNil(t, initContainer.SecurityContext.Privileged)
		assert.False(t, *initContainer.SecurityContext.Privileged)
		require.NotNil(t, initContainer.SecurityContext.AllowPrivilegeEscalation)
		assert.False(t, *initContainer.SecurityContext.AllowPrivilegeEscalation)
	})

	t.Run("resources come from the effective config", func(t *testing.T) {
		pod := newWebhookTestPod()

		createInitContainer(pod, map[string]struct{}{"{{ODIGOS_AGENTS_DIR}}/nodejs-community": {}}, config)

		require.Len(t, pod.Spec.InitContainers, 1)
		resources := pod.Spec.InitContainers[0].Resources
		assert.Equal(t, resource.MustParse("100m"), resources.Requests["cpu"])
		assert.Equal(t, resource.MustParse("200m"), resources.Limits["cpu"])
		assert.Equal(t, resource.MustParse("300Mi"), resources.Requests["memory"])
		assert.Equal(t, resource.MustParse("400Mi"), resources.Limits["memory"])
	})

	t.Run("is not added twice", func(t *testing.T) {
		pod := newWebhookTestPod()
		dirsToCopy := map[string]struct{}{"{{ODIGOS_AGENTS_DIR}}/nodejs-community": {}}

		createInitContainer(pod, dirsToCopy, config)
		createInitContainer(pod, dirsToCopy, config)

		assert.Len(t, pod.Spec.InitContainers, 1)
	})

	t.Run("user init containers are preserved", func(t *testing.T) {
		pod := newWebhookTestPod()
		pod.Spec.InitContainers = []corev1.Container{{Name: "user-init", Image: "user-image"}}

		createInitContainer(pod, map[string]struct{}{"{{ODIGOS_AGENTS_DIR}}/nodejs-community": {}}, config)

		require.Len(t, pod.Spec.InitContainers, 2)
		assert.Equal(t, "user-init", pod.Spec.InitContainers[0].Name)
		assert.Equal(t, k8sconsts.OdigosInitContainerName, pod.Spec.InitContainers[1].Name)
	})
}

func TestGetInitContainerImage(t *testing.T) {
	config := newWebhookTestConfig(common.K8sInitContainerMountMethod)
	config.ImagePrefix = "my-registry.io/odigos"

	t.Run("the image set at installation time wins", func(t *testing.T) {
		t.Setenv(k8sconsts.OdigosInitContainerEnvVarName, "my-registry.io/odigos/odigos-agents:v1.2.3")
		t.Setenv(commonconsts.OdigosVersionEnvVarName, "v1.0.0")

		assert.Equal(t, "my-registry.io/odigos/odigos-agents:v1.2.3", getInitContainerImage(config))
	})

	t.Run("falls back to the image prefix and the odigos version", func(t *testing.T) {
		t.Setenv(commonconsts.OdigosVersionEnvVarName, "v1.0.0")

		assert.Equal(t, "my-registry.io/odigos/odigos-agents:v1.0.0", getInitContainerImage(config))
	})
}

// ****************
// injectOdigos: distros that receive their configuration as env vars
// ****************

func TestInjectOdigosConfigAsEnvVars(t *testing.T) {
	ctx := webhookTestContext()
	req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{Namespace: webhookNamespace}}

	// php-community is the community distro that templates a static env var from a distro
	// parameter, reports its enabled signals as static otel env vars and receives its dynamic
	// configuration as env vars.
	newPhpIC := func() *odigosv1.InstrumentationConfig {
		ic := newWebhookTestIC()
		ic.Spec.Containers[0].OtelDistroName = "php-community"
		ic.Spec.Containers[0].DistroParams = map[string]string{distro.RuntimeVersionMajorMinorDistroParameterName: "8.3"}
		ic.Status.RuntimeDetailsByContainer[0].Language = common.PhpProgrammingLanguage
		return ic
	}

	t.Run("static env vars are templated with the distro params", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newPhpIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Equal(t, ":/var/odigos/php/8.3", webhookPodEnvValue(t, pod, 0, "PHP_INI_SCAN_DIR"))
		assert.Equal(t, "true", webhookPodEnvValue(t, pod, 0, "OTEL_PHP_AUTOLOAD_ENABLED"))
	})

	t.Run("a missing distro param fails the injection instead of templating a broken path", func(t *testing.T) {
		ic := newPhpIC()
		ic.Spec.Containers[0].DistroParams = nil
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)
		pod := newWebhookTestPod()

		require.Error(t, p.injectOdigos(ctx, pod, req))

		assert.False(t, webhookPodHasEnv(pod, 0, "PHP_INI_SCAN_DIR"))
	})

	t.Run("enabled signals are reported as static otel env vars", func(t *testing.T) {
		ic := newPhpIC()
		ic.Spec.Containers[0].Traces = &agentsignalconfig.AgentTracesConfig{}
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.Equal(t, "otlp", webhookPodEnvValue(t, pod, 0, "OTEL_TRACES_EXPORTER"))
		assert.Equal(t, "none", webhookPodEnvValue(t, pod, 0, "OTEL_METRICS_EXPORTER"))
		assert.Equal(t, "none", webhookPodEnvValue(t, pod, 0, "OTEL_LOGS_EXPORTER"))
	})

	t.Run("custom instrumentations are serialized into the container env", func(t *testing.T) {
		ic := newPhpIC()
		ic.Spec.Containers[0].Traces = &agentsignalconfig.AgentTracesConfig{
			CustomInstrumentations: &instrumentationrules.CustomInstrumentations{
				Php: []instrumentationrules.PhpCustomProbe{{ClassName: "MyClass", FunctionName: "myFunction"}},
			},
		}
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.JSONEq(t, `{"php":[{"className":"MyClass","functionName":"myFunction"}]}`,
			webhookPodEnvValue(t, pod, 0, k8sconsts.OdigosPhpAgentCustomInstrumentationsEnvVar))
	})

	t.Run("traces enabled without custom instrumentations means no env var", func(t *testing.T) {
		ic := newPhpIC()
		ic.Spec.Containers[0].Traces = &agentsignalconfig.AgentTracesConfig{}
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), ic)
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.False(t, webhookPodHasEnv(pod, 0, k8sconsts.OdigosPhpAgentCustomInstrumentationsEnvVar))
	})

	t.Run("a distro without an opamp client does not get the opamp env vars", func(t *testing.T) {
		p := newWebhookTestPodsWebhook(t, newWebhookTestConfig(common.K8sVirtualDeviceMountMethod), newPhpIC())
		pod := newWebhookTestPod()

		require.NoError(t, p.injectOdigos(ctx, pod, req))

		assert.False(t, webhookPodHasEnv(pod, 0, "ODIGOS_OPAMP_SERVER_HOST"))
		assert.False(t, webhookPodHasEnv(pod, 0, "ODIGOS_OPAMP_UNIX_SOCKET"))
	})
}
