package agentenabled

import (
	"context"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// ****************
// Setup helpers
// ****************

type syncTestSetup struct {
	ctx            context.Context
	scheme         *runtime.Scheme
	ns             *corev1.Namespace
	logger         *commonlogger.ContextLogger
	distroProvider *distros.Provider
}

func newSyncTestSetup() *syncTestSetup {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = odigosv1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = batchv1.AddToScheme(scheme)
	_ = argorolloutsv1alpha1.AddToScheme(scheme)

	getter, _ := distros.NewCommunityGetter()
	provider, _ := distros.NewProvider(distros.NewCommunityDefaulter(), getter)

	return &syncTestSetup{
		ctx:            context.Background(),
		scheme:         scheme,
		ns:             testutil.NewMockNamespace(),
		logger:         commonlogger.WrapLogr(logr.Discard()),
		distroProvider: provider,
	}
}

func (s *syncTestSetup) newFakeClient(objects ...client.Object) client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(s.scheme).
		WithObjects(objects...).
		Build()
}

// ****************
// Mock helpers
// ****************

// newHealthyPod creates a healthy running pod that matches a deployment's selector.
func newHealthyPod(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name": deploymentName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "test",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}
}

// newCrashLoopBackOffPodWithoutOdigosLabel creates a pod in CrashLoopBackOff WITHOUT odigos label.
func newCrashLoopBackOffPodWithoutOdigosLabel(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name": deploymentName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}
}

// newCrashLoopBackOffPodWithOdigosLabel creates a pod in CrashLoopBackOff WITH odigos label.
func newCrashLoopBackOffPodWithOdigosLabel(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            deploymentName,
				k8sconsts.OdigosAgentsMetaHashLabel: "some-hash",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}
}

// newWorkloadRefArgoRollout creates an Argo Rollout that takes its pod template from a
// workloadRef. argo-rollouts strips `spec.template` when it persists such a Rollout
// (RolloutSpec.MarshalJSON), so the object stored in the api server has no containers.
func newWorkloadRefArgoRollout(ns *corev1.Namespace, name string) *argorolloutsv1alpha1.Rollout {
	return &argorolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  ns.Name,
			Generation: 1,
		},
		Spec: argorolloutsv1alpha1.RolloutSpec{
			WorkloadRef: &argorolloutsv1alpha1.ObjectRef{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       name,
			},
		},
		Status: argorolloutsv1alpha1.RolloutStatus{
			ObservedGeneration: "1",
		},
	}
}

// newReadyNodeCollectorsGroup creates a node collectors group which satisfies the
// instrumentation prerequisites (ready and collecting at least one signal).
func newReadyNodeCollectorsGroup() *odigosv1alpha1.CollectorsGroup {
	return &odigosv1alpha1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: odigosv1alpha1.CollectorsGroupSpec{
			Role: odigosv1alpha1.CollectorsGroupRoleNodeCollector,
		},
		Status: odigosv1alpha1.CollectorsGroupStatus{
			Ready:           true,
			ReceiverSignals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}
}

// newMockCronJob creates a mock CronJob for testing.
func newMockCronJob(ns *corev1.Namespace, name string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.Name,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app.kubernetes.io/name": name},
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test",
								},
							},
						},
					},
				},
			},
		},
	}
}

// newMockJob creates a mock Job for testing.
func newMockJob(ns *corev1.Namespace, name string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": name},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
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
