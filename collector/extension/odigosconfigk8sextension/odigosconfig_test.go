package odigosconfigk8sextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// resourceWith builds a resource whose attributes are the given alternating key/value strings.
func resourceWith(kv ...string) pcommon.Resource {
	res := pcommon.NewResource()
	resourceAttrs(kv...).CopyTo(res.Attributes())
	return res
}

// deploymentConfigResource is what an instrumented OpenShift DeploymentConfig pod reports: the
// agent stamps the Odigos workload attributes, and the Kubernetes attributes processor adds
// k8s.deployment.name because a DeploymentConfig is backed by a ReplicationController the
// downstream enrichment resolves as a Deployment.
func deploymentConfigResource() pcommon.Resource {
	return resourceWith(
		string(semconv.K8SNamespaceNameKey), "prod",
		string(semconv.K8SContainerNameKey), "app",
		string(semconv.K8SDeploymentNameKey), "checkout",
		odigosWorkloadKindAttr, "DeploymentConfig",
		odigosWorkloadNameAttr, "checkout",
	)
}

// TestDeploymentConfigResourceResolvesItsOwnConfig is the end-to-end regression test for the
// OpenShift DeploymentConfig identity bug. The informer keys the cache off the InstrumentationConfig
// object name ("deploymentconfig-checkout"), while lookups key off resource attributes. When the
// lookup preferred the semconv attributes it produced "prod/Deployment/checkout/app" and every
// per-source feature — sampling, URL templatization, PII masking, data stream routing — silently
// fell back to defaults for every DeploymentConfig workload in the cluster.
func TestDeploymentConfigResourceResolvesItsOwnConfig(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	ic := newInstrumentationConfig("prod", "deploymentconfig-checkout", "app")
	ic.SetLabels(map[string]string{"odigos.io/data-stream-payments": "true"})
	o.handleInstrumentationConfig(ic)

	res := deploymentConfigResource()

	cacheKey, err := o.GetWorkloadCacheKey(res)
	require.NoError(t, err)
	require.Equal(t, "prod/DeploymentConfig/checkout/app", cacheKey)

	cfg, found := o.GetFromResource(res)
	require.True(t, found)
	require.Equal(t, "app", cfg.ContainerName)

	require.True(t, o.IsActiveSource(res))

	streams, found := o.GetDataStreamsForWorkload(res)
	require.True(t, found)
	require.Equal(t, []string{"payments"}, streams)
}

// The same resource must not resolve a same-named Deployment's configuration. Two workloads that
// differ only in kind are distinct Sources and can carry conflicting config.
func TestDeploymentConfigResourceDoesNotResolveASameNamedDeployment(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))

	res := deploymentConfigResource()

	_, found := o.GetFromResource(res)
	require.False(t, found)
	require.False(t, o.IsActiveSource(res))
	_, found = o.GetDataStreamsForWorkload(res)
	require.False(t, found)
}

// Every supported workload kind has to survive the round trip from the InstrumentationConfig object
// name the informer writes to the resource attributes a span carries. The two mappings live in
// different files and neither references the other, so nothing but a test keeps them aligned.
func TestWorkloadKindsRoundTripFromInformerToResource(t *testing.T) {
	tests := []struct {
		objectNamePrefix string
		resourceKind     string
	}{
		{"deployment", "Deployment"},
		{"daemonset", "DaemonSet"},
		{"statefulset", "StatefulSet"},
		{"cronjob", "CronJob"},
		{"job", "Job"},
		{"deploymentconfig", "DeploymentConfig"},
		{"rollout", "Rollout"},
		{"staticpod", "StaticPod"},
		{"namespace", "Namespace"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceKind, func(t *testing.T) {
			o, _ := newInformerTestExtension(t)
			o.handleInstrumentationConfig(newInstrumentationConfig("prod", tt.objectNamePrefix+"-checkout", "app"))

			res := resourceWith(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
				odigosWorkloadKindAttr, tt.resourceKind,
				odigosWorkloadNameAttr, "checkout",
			)

			cfg, found := o.GetFromResource(res)
			require.True(t, found)
			require.Equal(t, "app", cfg.ContainerName)
			require.True(t, o.IsActiveSource(res))
		})
	}
}

func TestGetFromResourceForUnidentifiableResource(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))

	res := resourceWith(string(semconv.K8SNamespaceNameKey), "prod")

	_, found := o.GetFromResource(res)
	require.False(t, found)
	require.False(t, o.IsActiveSource(res))

	_, err := o.GetWorkloadCacheKey(res)
	require.Error(t, err)

	_, _, err = o.GetWorkloadIdentityFromResource(res)
	require.Error(t, err)

	_, found = o.GetDataStreamsForWorkload(res)
	require.False(t, found)
}

// IsActiveSource works off the workload prefix, so it must answer for a resource that carries no
// container name at all — that is the whole reason the prefix lookup exists.
func TestIsActiveSourceWithoutAContainerName(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))

	res := resourceWith(
		string(semconv.K8SNamespaceNameKey), "prod",
		string(semconv.K8SDeploymentNameKey), "checkout",
	)
	require.True(t, o.IsActiveSource(res))

	// A container-level lookup still needs the container name.
	_, found := o.GetFromResource(res)
	require.False(t, found)
}

func TestGetWorkloadIdentityFromResource(t *testing.T) {
	o, _ := newInformerTestExtension(t)

	cacheKey, identity, err := o.GetWorkloadIdentityFromResource(deploymentConfigResource())
	require.NoError(t, err)
	require.Equal(t, "prod/DeploymentConfig/checkout/app", cacheKey)
	require.Equal(t, map[string]any{
		"k8s.namespace.name":   "prod",
		"k8s.container.name":   "app",
		"odigos.workload.kind": "DeploymentConfig",
		"odigos.workload.name": "checkout",
	}, identity.AsRaw())
}

// A processor that registers after the informer has synced must receive the current cache state,
// otherwise it starts with an empty view and applies default behavior until the next IC change,
// which for a stable cluster can be the full resync period.
func TestRegisterWorkloadConfigCacheCallbackBackfillsCurrentState(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app", "sidecar"))
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-frontend", "app"))

	cb := &recordingCallback{}
	o.RegisterWorkloadConfigCacheCallback(cb)

	require.ElementsMatch(t, []string{
		"set prod/Deployment/checkout/app",
		"set prod/Deployment/checkout/sidecar",
		"set prod/Deployment/frontend/app",
	}, cb.events)

	// The callback stays subscribed to later changes.
	cb.events = nil
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-frontend"))
	require.Equal(t, []string{"delete prod/Deployment/frontend/app"}, cb.events)
}

func TestUnregisterWorkloadConfigCacheCallback(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	cb := &recordingCallback{}
	o.RegisterWorkloadConfigCacheCallback(cb)

	o.UnregisterWorkloadConfigCacheCallback(cb)
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))

	require.Empty(t, cb.events)
}

// Shutdown must release the cache and the callbacks so a processor removed from the pipeline is
// not kept alive by the extension.
func TestShutdownReleasesCacheAndCallbacks(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	cb := &recordingCallback{}
	o.RegisterWorkloadConfigCacheCallback(cb)
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))
	cb.events = nil

	require.NoError(t, o.Shutdown(context.Background()))

	require.Empty(t, cacheKeys(o.cache))
	_, found := o.GetFromResource(resourceWith(
		string(semconv.K8SNamespaceNameKey), "prod",
		string(semconv.K8SContainerNameKey), "app",
		string(semconv.K8SDeploymentNameKey), "checkout",
	))
	require.False(t, found)

	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))
	require.Empty(t, cb.events)
}

// Outside a cluster the informer is never started and the cache stays empty, but the extension
// still starts and reports ready so the collector is not blocked.
func TestWaitForCacheSyncWithoutAnInformer(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	require.Nil(t, o.informerFactory)
	require.True(t, o.WaitForCacheSync(context.Background()))
}

func TestStartOutsideOfClusterDoesNotFail(t *testing.T) {
	// rest.InClusterConfig resolves the API server from the environment; clear it so the test does
	// not depend on where it runs.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	o, logs := newInformerTestExtension(t)

	require.NoError(t, o.Start(context.Background(), newMdatagenNopHost()))

	require.Nil(t, o.informerFactory)
	require.Len(t, logs.FilterMessage("not running in-cluster, instrumentation config cache will be empty").All(), 1)
	require.NoError(t, o.Shutdown(context.Background()))
}
