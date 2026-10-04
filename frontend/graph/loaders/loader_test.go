package loaders

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testOdigosNamespace = "odigos-test"

// fetchWorkloadManifests probes for OpenShift DeploymentConfigs through the package-level
// kube client, so it has to be set before any loader runs. An empty dynamic fake reports
// the resource as present with no objects, which keeps the probe out of the assertions.
func TestMain(m *testing.M) {
	dcGVR := schema.GroupVersionResource{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}
	kube.SetDefaultClient(&kube.Client{
		DynamicClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(),
			map[schema.GroupVersionResource]string{dcGVR: "DeploymentConfigList"},
		),
	})
	m.Run()
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, odigosv1.AddToScheme(s))
	return s
}

func effectiveConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: testOdigosNamespace,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: "ignoredNamespaces: []\n",
		},
	}
}

func workloadSource(name, namespace string, workloadName string, kind k8sconsts.WorkloadKind, asRegex bool) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: namespace,
				Kind:      kind,
				Name:      workloadName,
			},
			MatchWorkloadNameAsRegex: asRegex,
		},
	}
}

func deployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// newLoadersForTest builds a Loaders backed by the controller-runtime fake client.
// LoadWorkloadsWithFilter only needs the effective-config ConfigMap, the Source list,
// the InstrumentationConfig list and the workload manifest lists, all of which the
// fake client serves from the given objects.
func newLoadersForTest(t *testing.T, objs ...client.Object) *Loaders {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, testOdigosNamespace)

	c := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(objs...).
		Build()

	return NewLoaders(logr.Discard(), c)
}

func loadClusterWideWorkloadIds(t *testing.T, objs ...client.Object) []model.K8sWorkloadID {
	t.Helper()
	l := newLoadersForTest(t, objs...)
	require.NoError(t, l.LoadWorkloadsWithFilter(context.Background(), nil))
	return l.GetWorkloadIds()
}

// A Source with matchWorkloadNameAsRegex stores a regex pattern in spec.workload.name,
// so it does not identify any single workload and must not be reported as one.
func TestLoadWorkloadsWithFilter_GroupSourceIsNotAWorkload(t *testing.T) {
	ids := loadClusterWideWorkloadIds(t,
		effectiveConfigMap(),
		workloadSource("group-src", "default", "^payment-.*$", k8sconsts.WorkloadKindDeployment, true),
	)

	assert.Empty(t, ids, "a group (regex) source must not produce a workload")
}

// The workloads a group source matches are still listed, from their own manifests.
func TestLoadWorkloadsWithFilter_GroupSourceKeepsMatchedWorkloads(t *testing.T) {
	ids := loadClusterWideWorkloadIds(t,
		effectiveConfigMap(),
		workloadSource("group-src", "default", "^payment-.*$", k8sconsts.WorkloadKindDeployment, true),
		deployment("payment-api", "default"),
	)

	assert.Equal(t, []model.K8sWorkloadID{
		{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "payment-api"},
	}, ids)
}

// A per-workload Source whose workload manifest is gone is the "sources without
// workloads" case the function is documented to cover, so it stays visible.
func TestLoadWorkloadsWithFilter_SourceWithoutWorkloadIsListed(t *testing.T) {
	ids := loadClusterWideWorkloadIds(t,
		effectiveConfigMap(),
		workloadSource("orphan-src", "default", "checkout", k8sconsts.WorkloadKindDeployment, false),
	)

	assert.Equal(t, []model.K8sWorkloadID{
		{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "checkout"},
	}, ids)
}

// A workload that has both a manifest and a Source is reported exactly once.
func TestLoadWorkloadsWithFilter_WorkloadWithSourceIsNotDuplicated(t *testing.T) {
	ids := loadClusterWideWorkloadIds(t,
		effectiveConfigMap(),
		workloadSource("checkout-src", "default", "checkout", k8sconsts.WorkloadKindDeployment, false),
		deployment("checkout", "default"),
	)

	assert.Equal(t, []model.K8sWorkloadID{
		{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "checkout"},
	}, ids)
}

// A namespace-scoped Source instruments a whole namespace and is never a workload.
func TestLoadWorkloadsWithFilter_NamespaceSourceIsNotAWorkload(t *testing.T) {
	ids := loadClusterWideWorkloadIds(t,
		effectiveConfigMap(),
		workloadSource("ns-src", "default", "default", k8sconsts.WorkloadKindNamespace, false),
	)

	assert.Empty(t, ids, "a namespace source must not produce a workload")
}

// The mixed case: group sources dropped, per-workload sources and manifests kept.
func TestLoadWorkloadsWithFilter_MixedSources(t *testing.T) {
	ids := loadClusterWideWorkloadIds(t,
		effectiveConfigMap(),
		workloadSource("group-src", "default", "^payment-.*$", k8sconsts.WorkloadKindDeployment, true),
		workloadSource("orphan-src", "default", "checkout", k8sconsts.WorkloadKindDeployment, false),
		deployment("payment-api", "default"),
		&batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: "reporter", Namespace: "default"}},
	)

	assert.Equal(t, []model.K8sWorkloadID{
		{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "checkout"},
		{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "payment-api"},
		{Namespace: "default", Kind: model.K8sResourceKindCronJob, Name: "reporter"},
	}, ids)
}
