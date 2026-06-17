package sourceinstrumentation

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func TestInitiateConditionsSetLastTransitionTime(t *testing.T) {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
	workloadObj, err := workload.ObjectToWorkload(deployment)
	if err != nil {
		t.Fatalf("ObjectToWorkload: %v", err)
	}

	ic := &v1alpha1.InstrumentationConfig{}

	if !initiateRuntimeDetailsConditionIfMissing(ic, workloadObj) {
		t.Fatal("expected runtime details condition to be added")
	}
	if !initiateAgentEnabledConditionIfMissing(ic) {
		t.Fatal("expected agent enabled condition to be added")
	}

	for i, condition := range ic.Status.Conditions {
		if condition.LastTransitionTime.IsZero() {
			t.Fatalf("condition %d (%s) missing lastTransitionTime", i, condition.Type)
		}
	}
}

func TestInitiateRuntimeDetailsConditionIfMissingNoRunningPods(t *testing.T) {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
	}
	workloadObj, err := workload.ObjectToWorkload(deployment)
	if err != nil {
		t.Fatalf("ObjectToWorkload: %v", err)
	}

	ic := &v1alpha1.InstrumentationConfig{}
	if !initiateRuntimeDetailsConditionIfMissing(ic, workloadObj) {
		t.Fatal("expected runtime details condition to be added")
	}

	condition := ic.Status.Conditions[0]
	if condition.Reason != string(v1alpha1.RuntimeDetectionReasonNoRunningPods) {
		t.Fatalf("expected reason %q, got %q", v1alpha1.RuntimeDetectionReasonNoRunningPods, condition.Reason)
	}
	if condition.LastTransitionTime.IsZero() {
		t.Fatal("expected lastTransitionTime to be set")
	}
}

func TestInitiateAgentEnabledConditionIfMissingUsesNoRunningPodsReason(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{
		Status: v1alpha1.InstrumentationConfigStatus{
			Conditions: []metav1.Condition{
				{
					Type:               v1alpha1.RuntimeDetectionStatusConditionType,
					Status:             metav1.ConditionFalse,
					Reason:             string(v1alpha1.RuntimeDetectionReasonNoRunningPods),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	if !initiateAgentEnabledConditionIfMissing(ic) {
		t.Fatal("expected agent enabled condition to be added")
	}

	condition := ic.Status.Conditions[1]
	if condition.Reason != string(v1alpha1.AgentEnabledReasonRuntimeDetailsUnavailable) {
		t.Fatalf("expected reason %q, got %q", v1alpha1.AgentEnabledReasonRuntimeDetailsUnavailable, condition.Reason)
	}
	if condition.LastTransitionTime.IsZero() {
		t.Fatal("expected lastTransitionTime to be set")
	}
}

func TestInitiateRuntimeDetailsConditionIfMissingCronJob(t *testing.T) {
	controller := true
	ic := &v1alpha1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "batch/v1",
					Kind:       "CronJob",
					Controller: &controller,
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "CronJob",
		},
	}
	workloadObj, err := workload.ObjectToWorkload(cronJob)
	if err != nil {
		t.Fatalf("ObjectToWorkload: %v", err)
	}

	if !initiateRuntimeDetailsConditionIfMissing(ic, workloadObj) {
		t.Fatal("expected runtime details condition to be added")
	}

	condition := ic.Status.Conditions[0]
	if condition.Message != "Runtime detection pending Job to start" {
		t.Fatalf("unexpected message: %q", condition.Message)
	}
	if condition.LastTransitionTime.IsZero() {
		t.Fatal("expected lastTransitionTime to be set")
	}
}
