package graph

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	wuOdigosNamespace = "odigos-test-system"
	wuAppNamespace    = "checkout-ns"
	wuWorkloadName    = "checkout"

	// a distro that appears in loaders.isDistroExpectingInstrumentationInstances' allow list:
	// its agent reports InstrumentationInstances, so the health aggregation waits for them.
	wuReportingDistro = "golang-community"
	// a real distro that is deliberately absent from that allow list, so instances are never
	// expected for it and the workload must be reported as "unsupported" rather than "waiting".
	wuSilentDistro = "dotnet-community"
)

// wuContainer describes one container of a fixture pod in terms of the three inputs the health
// aggregation actually branches on. Each is produced by the real loader from a real pod manifest:
// distro comes from the ODIGOS_DISTRO_NAME env var, and ready from the container status.
type wuContainer struct {
	name string
	// empty means the container carries no odigos agent env var at all
	distro string
	ready  bool
}

type wuPod struct {
	name       string
	containers []wuContainer
}

type wuInstance struct {
	podName       string
	containerName string
	healthy       *bool
	message       string
	components    []v1alpha1.InstrumentationLibraryStatus
}

type wuFixture struct {
	pods      []wuPod
	instances []wuInstance
	// the zero value is accepted, so one constructor serves both the plain and the failing-read
	// cases. Nothing the priming step reads goes through List of pods or instances, so a
	// selective failure here only affects the aggregation under test.
	cacheFuncs interceptor.Funcs
}

func wuWorkloadID() model.K8sWorkloadID {
	return model.K8sWorkloadID{
		Namespace: wuAppNamespace,
		Kind:      model.K8sResourceKindDeployment,
		Name:      wuWorkloadName,
	}
}

func wuScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, v1alpha1.AddToScheme(s))
	return s
}

// wuEffectiveConfigMap is what loaders.LoadConfig reads. The body must be non-empty YAML:
// an empty body unmarshals into a nil *OdigosConfiguration and the pod loader then nil-derefs
// on the rollout settings.
func wuEffectiveConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: wuOdigosNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "configVersion: 1\n"},
	}
}

func wuDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: wuWorkloadName, Namespace: wuAppNamespace},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": wuWorkloadName}},
		},
	}
}

func wuPodObject(p wuPod) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.name,
			Namespace: wuAppNamespace,
			Labels:    map[string]string{"app": wuWorkloadName},
			// a Deployment's pods are owned by a ReplicaSet whose name carries a generated
			// suffix; that is how the loader resolves them back to the Deployment.
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       wuWorkloadName + "-7d4c8b5f9b",
			}},
		},
	}
	for _, c := range p.containers {
		container := corev1.Container{Name: c.name}
		if c.distro != "" {
			container.Env = []corev1.EnvVar{{Name: k8sconsts.OdigosEnvVarDistroName, Value: c.distro}}
		}
		pod.Spec.Containers = append(pod.Spec.Containers, container)
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:  c.name,
			Ready: c.ready,
		})
	}
	return pod
}

func wuInstanceObject(index int, ii wuInstance) *v1alpha1.InstrumentationInstance {
	return &v1alpha1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%d", ii.podName, ii.containerName, index),
			Namespace: wuAppNamespace,
			Labels: map[string]string{
				v1alpha1.OwnerPodNameLabel:      ii.podName,
				consts.InstrumentedAppNameLabel: workload.CalculateWorkloadRuntimeObjectName(wuWorkloadName, k8sconsts.WorkloadKindDeployment),
			},
		},
		Spec: v1alpha1.InstrumentationInstanceSpec{ContainerName: ii.containerName},
		Status: v1alpha1.InstrumentationInstanceStatus{
			Healthy:    ii.healthy,
			Message:    ii.message,
			Components: ii.components,
		},
	}
}

// wuLoaderContext primes a real *Loaders over a fake cluster holding the fixture and returns a
// context carrying it, the same way the gqlgen middleware does for a single-workload query.
// Going through the real loader rather than hand-building the cached pods keeps the distro
// allow list, the container-readiness mapping and the instance indexing in the contract.
func wuLoaderContext(t *testing.T, fixture wuFixture) context.Context {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, wuOdigosNamespace)

	objects := []client.Object{wuEffectiveConfigMap(), wuDeployment()}
	for _, p := range fixture.pods {
		objects = append(objects, wuPodObject(p))
	}
	for i, ii := range fixture.instances {
		objects = append(objects, wuInstanceObject(i, ii))
	}

	cacheClient := fake.NewClientBuilder().
		WithScheme(wuScheme(t)).
		WithObjects(objects...).
		WithInterceptorFuncs(fixture.cacheFuncs).
		Build()
	l := loaders.NewLoaders(logr.Discard(), cacheClient)

	ctx := context.Background()
	namespace, name := wuAppNamespace, wuWorkloadName
	kind := model.K8sResourceKindDeployment
	require.NoError(t, l.LoadWorkloadsWithFilter(ctx, &model.WorkloadFilter{
		Namespace: &namespace,
		Kind:      &kind,
		Name:      &name,
	}))

	return loaders.WithLoaders(ctx, l)
}

func wuBool(b bool) *bool { return &b }
