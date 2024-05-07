package datacollection

import (
	"os"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/stretchr/testify/assert"
)

const (
	mockNamespaceBase   = "test-namespace"
	mockDeploymentName  = "test-deployment"
	mockDaemonSetName   = "test-daemonset"
	mockStatefulSetName = "test-statefulset"
)

func NewMockNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewMockTestDeployment(ns *corev1.Namespace) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDeploymentName,
			Namespace: ns.GetName(),
		},
	}
}

func NewMockTestDaemonSet(ns *corev1.Namespace) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDaemonSetName,
			Namespace: ns.GetName(),
		},
	}
}

func NewMockTestStatefulSet(ns *corev1.Namespace) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockStatefulSetName,
			Namespace: ns.GetName(),
		},
	}
}

// givin a workload object (deployment, daemonset, statefulset) return a mock instrumented application
// with a single container with the GoProgrammingLanguage
func NewMockInstrumentedApplication(workloadObject client.Object) *odigosv1.InstrumentedApplication {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	return &odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GetRuntimeObjectName(workloadObject.GetName(), gvk.Kind),
			Namespace: workloadObject.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: workloadObject.GetName(),
					Kind: gvk.Kind,
				},
			},
		},
	}
}

func NewMockInstrumentedApplicationWoOwner(workloadObject client.Object) *odigosv1.InstrumentedApplication {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	return &odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GetRuntimeObjectName(workloadObject.GetName(), gvk.Kind),
			Namespace: workloadObject.GetNamespace(),
		},
	}
}

// Destination list must include a destination with LogsObservabilitySignal for the filelog to be configured
func NewMockDestinationList() *odigosv1.DestinationList {
	return &odigosv1.DestinationList{
		Items: []v1alpha1.Destination{
			{
				Spec: v1alpha1.DestinationSpec{
					Signals: []common.ObservabilitySignal{
						common.LogsObservabilitySignal,
					},
				},
			},
		},
	}
}

func openTestData(t *testing.T, path string) string {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to open %s", path)
	}
	return string(want)
}

func TestGetConfigMapData(t *testing.T) {
	want := openTestData(t, "testdata/logs_included.yaml")

	ns := NewMockNamespace("default")
	ns2 := NewMockNamespace("other-namespace")

	items := []v1alpha1.InstrumentedApplication{
		*NewMockInstrumentedApplication(NewMockTestDeployment(ns)),
		*NewMockInstrumentedApplication(NewMockTestDaemonSet(ns)),
		*NewMockInstrumentedApplication(NewMockTestStatefulSet(ns2)),
		*NewMockInstrumentedApplicationWoOwner(NewMockTestDeployment(ns2)),
	}

	got, err := getConfigMapData(
		&v1alpha1.InstrumentedApplicationList{
			Items: items,
		},
		NewMockDestinationList(),
		&v1alpha1.ProcessorList{},
	)

	assert.Equal(t, err, nil)
	assert.Equal(t, want, got)
}
