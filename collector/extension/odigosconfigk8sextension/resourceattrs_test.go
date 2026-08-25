package odigosconfigk8sextension

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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

// Attribute keys are spelled out here on purpose: asserting with the same constant the production
// code reads would make these tests follow a rename instead of pinning the wire format.
const (
	odigosWorkloadKindAttr = "odigos.workload.kind"
	odigosWorkloadNameAttr = "odigos.workload.name"
	argoRolloutNameAttr    = "k8s.argoproj.rollout.name"
)

// resourceAttrs builds a resource attribute map from alternating key/value strings.
func resourceAttrs(kv ...string) pcommon.Map {
	attrs := pcommon.NewMap()
	for i := 0; i < len(kv); i += 2 {
		attrs.PutStr(kv[i], kv[i+1])
	}
	return attrs
}

// TestGetKindAndNameFromSemconvAttributes pins the semconv fallback used for workloads whose
// resource does not carry the Odigos workload attributes (e.g. spans from a collector-only
// pipeline). The Kind produced here has to match the Kind the informer writes into the cache key,
// so a change to this mapping silently breaks every per-source config lookup.
func TestGetKindAndNameFromSemconvAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    pcommon.Map
		wantKind string
		wantName string
	}{
		{
			name:     "deployment",
			attrs:    resourceAttrs(string(semconv.K8SDeploymentNameKey), "checkout"),
			wantKind: "Deployment",
			wantName: "checkout",
		},
		{
			name:     "statefulset",
			attrs:    resourceAttrs(string(semconv.K8SStatefulSetNameKey), "postgres"),
			wantKind: "StatefulSet",
			wantName: "postgres",
		},
		{
			name:     "daemonset",
			attrs:    resourceAttrs(string(semconv.K8SDaemonSetNameKey), "fluentd"),
			wantKind: "DaemonSet",
			wantName: "fluentd",
		},
		{
			name:     "cronjob",
			attrs:    resourceAttrs(string(semconv.K8SCronJobNameKey), "report"),
			wantKind: "CronJob",
			wantName: "report",
		},
		{
			name:     "argo rollout",
			attrs:    resourceAttrs(argoRolloutNameAttr, "canary"),
			wantKind: "Rollout",
			wantName: "canary",
		},
		{
			// k8s.job.name was removed from the lookup: a bare Job pod resource resolves to no
			// workload rather than to a Job that is not a Source kind.
			name:     "job name alone resolves to nothing",
			attrs:    resourceAttrs(string(semconv.K8SJobNameKey), "migrate"),
			wantKind: "",
			wantName: "",
		},
		{
			// A CronJob's pods carry k8s.job.name for the run and k8s.cronjob.name for the owner.
			// The config belongs to the CronJob Source, so the job attribute must not win.
			name: "cronjob pod carrying its job name resolves to the cronjob",
			attrs: resourceAttrs(
				string(semconv.K8SJobNameKey), "report-28472400",
				string(semconv.K8SCronJobNameKey), "report",
			),
			wantKind: "CronJob",
			wantName: "report",
		},
		{
			name: "first matching pair wins",
			attrs: resourceAttrs(
				string(semconv.K8SStatefulSetNameKey), "postgres",
				string(semconv.K8SDeploymentNameKey), "checkout",
			),
			wantKind: "Deployment",
			wantName: "checkout",
		},
		{
			name:     "no workload attribute",
			attrs:    resourceAttrs(string(semconv.K8SPodNameKey), "checkout-abc-123"),
			wantKind: "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, name := getKindAndName(tt.attrs)
			require.Equal(t, tt.wantKind, kind)
			require.Equal(t, tt.wantName, name)
		})
	}
}

// A workload name attribute of the wrong type must be skipped rather than resolving the workload
// to an empty name, which would produce a cache key that can never match an informer entry. Both
// lookups have to skip it or the cache key and the reported identity describe different workloads.
func TestNonStringSemconvAttributeIsSkipped(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "prod")
	attrs.PutStr(string(semconv.K8SContainerNameKey), "app")
	attrs.PutInt(string(semconv.K8SDeploymentNameKey), 42)
	attrs.PutStr(string(semconv.K8SStatefulSetNameKey), "postgres")

	kind, name := getKindAndName(attrs)
	require.Equal(t, "StatefulSet", kind)
	require.Equal(t, "postgres", name)

	require.Equal(t, map[string]any{
		"k8s.namespace.name":   "prod",
		"k8s.container.name":   "app",
		"k8s.statefulset.name": "postgres",
	}, identifyingResourceAttributes(attrs).AsRaw())
}

// TestGetKindAndNamePrefersOdigosWorkloadAttributes covers the OpenShift DeploymentConfig fix:
// DeploymentConfig pods carry k8s.deployment.name, so resolving identity from semconv first
// labelled them as Deployments and their per-source config never resolved.
func TestGetKindAndNamePrefersOdigosWorkloadAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    pcommon.Map
		wantKind string
		wantName string
	}{
		{
			name: "openshift deployment config also carries the deployment name",
			attrs: resourceAttrs(
				string(semconv.K8SDeploymentNameKey), "checkout",
				odigosWorkloadKindAttr, "DeploymentConfig",
				odigosWorkloadNameAttr, "checkout",
			),
			wantKind: "DeploymentConfig",
			wantName: "checkout",
		},
		{
			name: "job is only reachable through the odigos attributes",
			attrs: resourceAttrs(
				string(semconv.K8SJobNameKey), "migrate",
				odigosWorkloadKindAttr, "Job",
				odigosWorkloadNameAttr, "migrate",
			),
			wantKind: "Job",
			wantName: "migrate",
		},
		{
			// The owning CronJob is the Source; the odigos attributes name it directly.
			name: "cronjob pod",
			attrs: resourceAttrs(
				string(semconv.K8SJobNameKey), "report-28472400",
				string(semconv.K8SCronJobNameKey), "report",
				odigosWorkloadKindAttr, "CronJob",
				odigosWorkloadNameAttr, "report",
			),
			wantKind: "CronJob",
			wantName: "report",
		},
		{
			name: "odigos attributes win over a matching deployment",
			attrs: resourceAttrs(
				string(semconv.K8SDeploymentNameKey), "checkout",
				odigosWorkloadKindAttr, "Deployment",
				odigosWorkloadNameAttr, "checkout",
			),
			wantKind: "Deployment",
			wantName: "checkout",
		},
		{
			name: "odigos attributes are used when no semconv attribute is present",
			attrs: resourceAttrs(
				odigosWorkloadKindAttr, "StaticPod",
				odigosWorkloadNameAttr, "etcd",
			),
			wantKind: "StaticPod",
			wantName: "etcd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, name := getKindAndName(tt.attrs)
			require.Equal(t, tt.wantKind, kind)
			require.Equal(t, tt.wantName, name)
		})
	}
}

// Both Odigos attributes are required together. Taking only one of them and leaving the other
// empty would build a cache key that matches nothing, so a half-populated resource has to fall
// back to the semconv lookup.
func TestGetKindAndNameRequiresBothOdigosAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    pcommon.Map
		wantKind string
		wantName string
	}{
		{
			name: "kind only falls back to semconv",
			attrs: resourceAttrs(
				odigosWorkloadKindAttr, "DeploymentConfig",
				string(semconv.K8SDeploymentNameKey), "checkout",
			),
			wantKind: "Deployment",
			wantName: "checkout",
		},
		{
			name: "name only falls back to semconv",
			attrs: resourceAttrs(
				odigosWorkloadNameAttr, "checkout",
				string(semconv.K8SDeploymentNameKey), "checkout",
			),
			wantKind: "Deployment",
			wantName: "checkout",
		},
		{
			name:     "kind only with no semconv attribute resolves to nothing",
			attrs:    resourceAttrs(odigosWorkloadKindAttr, "DeploymentConfig"),
			wantKind: "",
			wantName: "",
		},
		{
			name:     "name only with no semconv attribute resolves to nothing",
			attrs:    resourceAttrs(odigosWorkloadNameAttr, "checkout"),
			wantKind: "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, name := getKindAndName(tt.attrs)
			require.Equal(t, tt.wantKind, kind)
			require.Equal(t, tt.wantName, name)
		})
	}
}

// TestIdentifyingResourceAttributes pins the exact attribute set stamped on telemetry generated for
// a workload. Emitting k8s.deployment.name for a DeploymentConfig makes consumers reconstruct the
// wrong Source, so the identity has to carry the same workload attributes the cache key was
// derived from and nothing else.
func TestIdentifyingResourceAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attrs pcommon.Map
		want  map[string]any
	}{
		{
			name: "deployment identified by semconv",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
				string(semconv.K8SPodNameKey), "checkout-abc-123",
				string(semconv.K8SDeploymentNameKey), "checkout",
			),
			want: map[string]any{
				"k8s.namespace.name":  "prod",
				"k8s.container.name":  "app",
				"k8s.deployment.name": "checkout",
			},
		},
		{
			name: "deployment config does not leak the deployment name",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
				string(semconv.K8SDeploymentNameKey), "checkout",
				odigosWorkloadKindAttr, "DeploymentConfig",
				odigosWorkloadNameAttr, "checkout",
			),
			want: map[string]any{
				"k8s.namespace.name":   "prod",
				"k8s.container.name":   "app",
				"odigos.workload.kind": "DeploymentConfig",
				"odigos.workload.name": "checkout",
			},
		},
		{
			// A half-populated pair must fall back to semconv rather than emit an empty attribute:
			// the identity has to describe the same workload the cache key was built from.
			name: "odigos kind without a name falls back to semconv",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
				string(semconv.K8SDeploymentNameKey), "checkout",
				odigosWorkloadKindAttr, "DeploymentConfig",
			),
			want: map[string]any{
				"k8s.namespace.name":  "prod",
				"k8s.container.name":  "app",
				"k8s.deployment.name": "checkout",
			},
		},
		{
			name: "odigos name without a kind falls back to semconv",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
				string(semconv.K8SDeploymentNameKey), "checkout",
				odigosWorkloadNameAttr, "checkout",
			),
			want: map[string]any{
				"k8s.namespace.name":  "prod",
				"k8s.container.name":  "app",
				"k8s.deployment.name": "checkout",
			},
		},
		{
			name: "only one workload attribute is emitted",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
				string(semconv.K8SDeploymentNameKey), "checkout",
				string(semconv.K8SStatefulSetNameKey), "postgres",
			),
			want: map[string]any{
				"k8s.namespace.name":  "prod",
				"k8s.container.name":  "app",
				"k8s.deployment.name": "checkout",
			},
		},
		{
			name: "unidentifiable workload still reports namespace and container",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
			),
			want: map[string]any{
				"k8s.namespace.name": "prod",
				"k8s.container.name": "app",
			},
		},
		{
			name:  "no attributes at all",
			attrs: pcommon.NewMap(),
			want:  map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, identifyingResourceAttributes(tt.attrs).AsRaw())
		})
	}
}

// The cache key and the identifying attributes are produced by two separate lookups; if they ever
// disagree a workload is looked up under one identity and reported under another.
func TestWorkloadIdentityFromResourceAttributesForDeploymentConfig(t *testing.T) {
	attrs := resourceAttrs(
		string(semconv.K8SNamespaceNameKey), "prod",
		string(semconv.K8SContainerNameKey), "app",
		string(semconv.K8SDeploymentNameKey), "checkout",
		odigosWorkloadKindAttr, "DeploymentConfig",
		odigosWorkloadNameAttr, "checkout",
	)

	cacheKey, identity, err := workloadIdentityFromResourceAttributes(attrs)
	require.NoError(t, err)
	require.Equal(t, "prod/DeploymentConfig/checkout/app", cacheKey)
	require.Equal(t, map[string]any{
		"k8s.namespace.name":   "prod",
		"k8s.container.name":   "app",
		"odigos.workload.kind": "DeploymentConfig",
		"odigos.workload.name": "checkout",
	}, identity.AsRaw())
}

func TestWorkloadKeyFromResourceAttributesRequiresEveryPart(t *testing.T) {
	tests := []struct {
		name  string
		attrs pcommon.Map
	}{
		{
			name: "missing namespace",
			attrs: resourceAttrs(
				string(semconv.K8SContainerNameKey), "app",
				string(semconv.K8SDeploymentNameKey), "checkout",
			),
		},
		{
			name: "missing container",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SDeploymentNameKey), "checkout",
			),
		},
		{
			name: "missing workload",
			attrs: resourceAttrs(
				string(semconv.K8SNamespaceNameKey), "prod",
				string(semconv.K8SContainerNameKey), "app",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := workloadKeyFromResourceAttributes(tt.attrs)
			require.EqualError(t, err, "workload info cannot be calculated from the resource attributes")
			require.Empty(t, key)
		})
	}
}

// The workload-level prefix is what IsActiveSource and GetDataStreamsForWorkload match against the
// cache index, so it must carry the trailing slash and must not require a container name.
func TestWorkloadContainerKeyFromResourceAttributes(t *testing.T) {
	t.Run("does not require a container name", func(t *testing.T) {
		attrs := resourceAttrs(
			string(semconv.K8SNamespaceNameKey), "prod",
			string(semconv.K8SDeploymentNameKey), "checkout",
		)

		key, err := workloadContainerKeyFromResourceAttributes(attrs)
		require.NoError(t, err)
		require.Equal(t, "prod/Deployment/checkout/", key)
	})

	t.Run("prefers the odigos workload attributes", func(t *testing.T) {
		attrs := resourceAttrs(
			string(semconv.K8SNamespaceNameKey), "prod",
			string(semconv.K8SDeploymentNameKey), "checkout",
			odigosWorkloadKindAttr, "DeploymentConfig",
			odigosWorkloadNameAttr, "checkout",
		)

		key, err := workloadContainerKeyFromResourceAttributes(attrs)
		require.NoError(t, err)
		require.Equal(t, "prod/DeploymentConfig/checkout/", key)
	})

	t.Run("unidentifiable workload", func(t *testing.T) {
		attrs := resourceAttrs(string(semconv.K8SNamespaceNameKey), "prod")

		key, err := workloadContainerKeyFromResourceAttributes(attrs)
		require.EqualError(t, err, "workload info cannot be calculated from the resource attributes")
		require.Empty(t, key)
	})
}
