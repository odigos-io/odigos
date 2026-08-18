package odigosconfigk8sextension

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/odigos-io/odigos/common/consts"
)

func TestWorkloadIdentityFromResourceAttributes(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "default")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "checkout")
	attrs.PutStr(string(semconv.K8SContainerNameKey), "app")

	cacheKey, identityAttrs, err := workloadIdentityFromResourceAttributes(attrs)
	require.NoError(t, err)
	require.Equal(t, "default/Deployment/checkout/app", cacheKey)

	namespace, ok := identityAttrs.Get(string(semconv.K8SNamespaceNameKey))
	require.True(t, ok)
	require.Equal(t, "default", namespace.Str())

	deployment, ok := identityAttrs.Get(string(semconv.K8SDeploymentNameKey))
	require.True(t, ok)
	require.Equal(t, "checkout", deployment.Str())

	container, ok := identityAttrs.Get(string(semconv.K8SContainerNameKey))
	require.True(t, ok)
	require.Equal(t, "app", container.Str())
}

func TestWorkloadIdentityFromResourceAttributes_MissingAttributes(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "default")

	_, _, err := workloadIdentityFromResourceAttributes(attrs)
	require.Error(t, err)
}

// The keys the pods webhook injects into OTEL_RESOURCE_ATTRIBUTES, and the keys odiglet sets for
// eBPF instrumented processes, must resolve to the same key the InstrumentationConfig is cached
// under, otherwise every processor that reads per-source config silently skips the workload.
func TestWorkloadKeyFromResourceAttributes_MatchesInstrumentationConfigKey(t *testing.T) {
	for _, tc := range []struct {
		name string
		// the InstrumentationConfig object name for the source
		icName string
		attrs  map[string]string
	}{
		{
			name:   "deployment",
			icName: "deployment-checkout",
			attrs: map[string]string{
				string(semconv.K8SDeploymentNameKey): "checkout",
				consts.OdigosWorkloadKindAttribute:   "Deployment",
				consts.OdigosWorkloadNameAttribute:   "checkout",
				string(semconv.K8SReplicaSetNameKey): "checkout-5d9f8c7b6d",
			},
		},
		{
			name:   "statefulset",
			icName: "statefulset-postgres",
			attrs: map[string]string{
				string(semconv.K8SStatefulSetNameKey): "postgres",
				consts.OdigosWorkloadKindAttribute:    "StatefulSet",
				consts.OdigosWorkloadNameAttribute:    "postgres",
			},
		},
		{
			name:   "daemonset",
			icName: "daemonset-fluentd",
			attrs: map[string]string{
				string(semconv.K8SDaemonSetNameKey): "fluentd",
				consts.OdigosWorkloadKindAttribute:  "DaemonSet",
				consts.OdigosWorkloadNameAttribute:  "fluentd",
			},
		},
		{
			// a CronJob pod is owned by the Job of the current run, so the webhook adds
			// k8s.job.name (of the Job) next to k8s.cronjob.name (of the Source)
			name:   "cronjob also carries the per-run job name",
			icName: "cronjob-backup",
			attrs: map[string]string{
				string(semconv.K8SCronJobNameKey):  "backup",
				string(semconv.K8SJobNameKey):      "backup-28812345",
				consts.OdigosWorkloadKindAttribute: "CronJob",
				consts.OdigosWorkloadNameAttribute: "backup",
			},
		},
		{
			// OpenShift DeploymentConfig reuses k8s.deployment.name, so only the
			// odigos.workload.* pair tells it apart from a Deployment
			name:   "deploymentconfig reuses the deployment semconv key",
			icName: "deploymentconfig-frontend",
			attrs: map[string]string{
				string(semconv.K8SDeploymentNameKey): "frontend",
				consts.OdigosWorkloadKindAttribute:   "DeploymentConfig",
				consts.OdigosWorkloadNameAttribute:   "frontend",
			},
		},
		{
			name:   "argo rollout",
			icName: "rollout-payments",
			attrs: map[string]string{
				k8SArgoRolloutNameAttribute:          "payments",
				string(semconv.K8SReplicaSetNameKey): "payments-6f7c9d4b8",
				consts.OdigosWorkloadKindAttribute:   "Rollout",
				consts.OdigosWorkloadNameAttribute:   "payments",
			},
		},
		{
			// odiglet sets no semconv workload key for a static pod
			name:   "static pod",
			icName: "staticpod-kube-apiserver-node1",
			attrs: map[string]string{
				consts.OdigosWorkloadKindAttribute: "StaticPod",
				consts.OdigosWorkloadNameAttribute: "kube-apiserver-node1",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			attrs.PutStr(string(semconv.K8SNamespaceNameKey), "prod")
			attrs.PutStr(string(semconv.K8SContainerNameKey), "app")
			for k, v := range tc.attrs {
				attrs.PutStr(k, v)
			}

			got, err := workloadKeyFromResourceAttributes(attrs)
			require.NoError(t, err)
			require.Equal(t, instrumentationConfigCacheKey(t, tc.icName, "prod", "app"), got)
		})
	}
}

// Telemetry that never passed through the Odigos webhook or odiglet (e.g. enriched only by the
// k8sattributes processor) has to keep resolving from the semconv keys alone.
func TestWorkloadKeyFromResourceAttributes_SemconvOnly(t *testing.T) {
	for _, tc := range []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{
			name:  "deployment",
			attrs: map[string]string{string(semconv.K8SDeploymentNameKey): "checkout"},
			want:  "prod/Deployment/checkout/app",
		},
		{
			name: "cronjob wins over the per-run job name",
			attrs: map[string]string{
				string(semconv.K8SCronJobNameKey): "backup",
				string(semconv.K8SJobNameKey):     "backup-28812345",
			},
			want: "prod/CronJob/backup/app",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			attrs.PutStr(string(semconv.K8SNamespaceNameKey), "prod")
			attrs.PutStr(string(semconv.K8SContainerNameKey), "app")
			for k, v := range tc.attrs {
				attrs.PutStr(k, v)
			}

			got, err := workloadKeyFromResourceAttributes(attrs)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// A partial odigos.workload.* pair must not shadow the semconv keys.
func TestWorkloadKeyFromResourceAttributes_PartialOdigosPair(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "prod")
	attrs.PutStr(string(semconv.K8SContainerNameKey), "app")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "checkout")
	attrs.PutStr(consts.OdigosWorkloadKindAttribute, "Deployment")

	got, err := workloadKeyFromResourceAttributes(attrs)
	require.NoError(t, err)
	require.Equal(t, "prod/Deployment/checkout/app", got)
}

// instrumentationConfigCacheKey derives the cache key the informer stores a container's config
// under, from the InstrumentationConfig object name, the way handleInstrumentationConfig does.
func instrumentationConfigCacheKey(t *testing.T, icName, namespace, containerName string) string {
	t.Helper()
	u := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": icName, "namespace": namespace},
	}}
	wk, ok := workloadKeyFromObject(u)
	require.True(t, ok, "workloadKeyFromObject failed for %q", icName)
	return k8sSourceKey(wk.Namespace, wk.Kind, wk.Name, containerName)
}
