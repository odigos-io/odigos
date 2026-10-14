package controllers

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkloadTransformKeepsOdigosNamespaceFull(t *testing.T) {
	t.Parallel()

	odigosNs := "odigos-system"
	transform := workloadTransformFunc(odigosNs)

	full := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos-gateway", Namespace: odigosNs},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "gateway",
						Image: "registry.odigos.io/odigos-collector",
					}},
				},
			},
		},
	}

	got, err := transform(full)
	require.NoError(t, err)
	dep := got.(*appsv1.Deployment)
	require.Equal(t, "registry.odigos.io/odigos-collector", dep.Spec.Template.Spec.Containers[0].Image)
}

func TestWorkloadTransformStripsOtherNamespaces(t *testing.T) {
	t.Parallel()

	transform := workloadTransformFunc("odigos-system")

	user := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "frontend", Namespace: "app"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: "example/frontend:1",
					}},
				},
			},
		},
	}

	got, err := transform(user)
	require.NoError(t, err)
	dep := got.(*appsv1.Deployment)
	require.Equal(t, "app", dep.Spec.Template.Spec.Containers[0].Name)
	require.Empty(t, dep.Spec.Template.Spec.Containers[0].Image)
}

func TestPodTransformKeepsOdigosNamespacePodIP(t *testing.T) {
	t.Parallel()

	transform := podTransformFunc("odigos-system")
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos-gateway-0", Namespace: "odigos-system"},
		Status:     corev1.PodStatus{PodIP: "10.0.0.8", Phase: corev1.PodRunning},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "gateway", Image: "collector"}},
		},
	}

	got, err := transform(pod)
	require.NoError(t, err)
	require.Equal(t, "10.0.0.8", got.(*corev1.Pod).Status.PodIP)
	require.Equal(t, "collector", got.(*corev1.Pod).Spec.Containers[0].Image)
}
