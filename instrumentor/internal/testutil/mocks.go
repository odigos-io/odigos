package testutil

import (
	"fmt"

	"github.com/google/uuid"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	mockNamespaceBase   = "test-namespace"
	mockDeploymentName  = "test-deployment"
	mockDaemonSetName   = "test-daemonset"
	mockStatefulSetName = "test-statefulset"
)

func NewOdigosSystemNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-system",
		},
	}
}

func NewMockNamespace() *corev1.Namespace {
	name := generateUUIDNamespace(mockNamespaceBase)
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
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "test-dep"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "test-dep"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
				},
			},
		},
	}
}

func NewMockTestDaemonSet(ns *corev1.Namespace) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDaemonSetName,
			Namespace: ns.GetName(),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "test-ds"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "test-ds"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
				},
			},
		},
	}
}

func NewMockTestStatefulSet(ns *corev1.Namespace) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockStatefulSetName,
			Namespace: ns.GetName(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "test-ss"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "test-ss"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
				},
			},
		},
	}
}

// givin a workload object (deployment, daemonset, statefulset) return a mock instrumented application
// with a single container with the GoProgrammingLanguage
func NewMockInstrumentedApplication(workloadObject client.Object) *odigosv1.InstrumentedApplication {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	return &odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.GetRuntimeObjectName(workloadObject.GetName(), gvk.Kind),
			Namespace: workloadObject.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Name:       workloadObject.GetName(),
					UID:        workloadObject.GetUID(),
				},
			},
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			RuntimeDetails: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test",
					Language:      common.GoProgrammingLanguage,
				},
			},
		},
	}
}

func NewMockDataCollection() *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-data-collection",
			Namespace: consts.DefaultNamespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleNodeCollector,
		},
	}
}

func NewMockOdigosConfig() *odigosv1.OdigosConfiguration {
	return &odigosv1.OdigosConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-config",
			Namespace: consts.DefaultNamespace,
		},
		Spec: odigosv1.OdigosConfigurationSpec{
			DefaultSDKs: map[common.ProgrammingLanguage]common.OtelSdk{
				common.PythonProgrammingLanguage: common.OtelSdkNativeCommunity,
				common.GoProgrammingLanguage:     common.OtelSdkNativeCommunity,
			},
		},
	}
}

// this helps to avoid the "already exists" error when creating a new namespace.
// it promotes test isolation and avoid conflicts between tests.
func generateUUIDNamespace(baseName string) string {
	return fmt.Sprintf("%s-%s", baseName, uuid.New().String())
}
