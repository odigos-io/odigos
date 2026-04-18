package kube

import (
	"reflect"
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBuildCacheByObjectConfig_DisablesUnsafeDeepCopyForMutatedTypes(t *testing.T) {
	oldArgo := IsArgoRolloutAvailable
	oldOpenShift := IsOpenShiftDeploymentConfigAvailable
	IsArgoRolloutAvailable = true
	IsOpenShiftDeploymentConfigAvailable = true
	t.Cleanup(func() {
		IsArgoRolloutAvailable = oldArgo
		IsOpenShiftDeploymentConfigAvailable = oldOpenShift
	})

	byObject := buildCacheByObjectConfig("odigos-system")

	assertUnsafeDeepCopy(t, byObjectForType(t, byObject, &corev1.ConfigMap{}), false)
	assertUnsafeDeepCopy(t, byObjectForType(t, byObject, &odigosv1.Source{}), false)
	assertUnsafeDeepCopy(t, byObjectForType(t, byObject, &odigosv1.InstrumentationConfig{}), false)
	assertUnsafeDeepCopy(t, byObjectForType(t, byObject, &odigosv1.InstrumentationInstance{}), false)
	assertUnsafeDeepCopy(t, byObjectForType(t, byObject, &odigosv1.Destination{}), false)
	assertUnsafeDeepCopy(t, byObjectForType(t, byObject, &odigosv1.Sampling{}), false)
}

func TestBuildCacheByObjectConfig_KeepsUnsafeDeepCopyForReadOnlyHotTypes(t *testing.T) {
	oldArgo := IsArgoRolloutAvailable
	oldOpenShift := IsOpenShiftDeploymentConfigAvailable
	IsArgoRolloutAvailable = true
	IsOpenShiftDeploymentConfigAvailable = true
	t.Cleanup(func() {
		IsArgoRolloutAvailable = oldArgo
		IsOpenShiftDeploymentConfigAvailable = oldOpenShift
	})

	byObject := buildCacheByObjectConfig("odigos-system")

	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &corev1.Pod{}))
	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &corev1.Namespace{}))
	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &appsv1.Deployment{}))
	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &appsv1.DaemonSet{}))
	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &appsv1.StatefulSet{}))
	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &batchv1.CronJob{}))
	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &argorolloutsv1alpha1.Rollout{}))
	assertUnsafeDeepCopyUnset(t, byObjectForType(t, byObject, &openshiftappsv1.DeploymentConfig{}))
}

func byObjectForType(t *testing.T, byObject map[client.Object]cache.ByObject, target client.Object) cache.ByObject {
	t.Helper()
	targetType := reflect.TypeOf(target)
	for obj, cfg := range byObject {
		if reflect.TypeOf(obj) == targetType {
			return cfg
		}
	}
	t.Fatalf("missing cache config for %v", targetType)
	return cache.ByObject{}
}

func assertUnsafeDeepCopy(t *testing.T, cfg cache.ByObject, expected bool) {
	t.Helper()
	if cfg.UnsafeDisableDeepCopy == nil {
		t.Fatalf("expected UnsafeDisableDeepCopy=%v, got nil", expected)
	}
	if *cfg.UnsafeDisableDeepCopy != expected {
		t.Fatalf("expected UnsafeDisableDeepCopy=%v, got %v", expected, *cfg.UnsafeDisableDeepCopy)
	}
}

func assertUnsafeDeepCopyUnset(t *testing.T, cfg cache.ByObject) {
	t.Helper()
	if cfg.UnsafeDisableDeepCopy != nil {
		t.Fatalf("expected UnsafeDisableDeepCopy=nil, got %v", *cfg.UnsafeDisableDeepCopy)
	}
}
