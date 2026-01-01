package workload

import (
	"errors"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Workload interface {
	client.Object
	AvailableReplicas() int32
	PodSpec() *corev1.PodSpec
}

// compile time check for interface implementation
var _ Workload = &DeploymentWorkload{}
var _ Workload = &DaemonSetWorkload{}
var _ Workload = &StatefulSetWorkload{}
var _ Workload = &StaticPodWorkload{}
var _ Workload = &CronJobWorkloadV1{}
var _ Workload = &CronJobWorkloadBeta{}
var _ Workload = &DeploymentConfigWorkload{}
var _ Workload = &ArgoRolloutWorkload{}

type DeploymentWorkload struct {
	*v1.Deployment
}

func (d *DeploymentWorkload) AvailableReplicas() int32 {
	return d.Status.AvailableReplicas
}

func (d *DeploymentWorkload) PodSpec() *corev1.PodSpec {
	return &d.Spec.Template.Spec
}

type DaemonSetWorkload struct {
	*v1.DaemonSet
}

func (d *DaemonSetWorkload) AvailableReplicas() int32 {
	return d.Status.NumberReady
}

func (d *DaemonSetWorkload) PodSpec() *corev1.PodSpec {
	return &d.Spec.Template.Spec
}

type StatefulSetWorkload struct {
	*v1.StatefulSet
}

func (s *StatefulSetWorkload) AvailableReplicas() int32 {
	return s.Status.ReadyReplicas
}

func (s *StatefulSetWorkload) PodSpec() *corev1.PodSpec {
	return &s.Spec.Template.Spec
}

type StaticPodWorkload struct {
	*corev1.Pod
}

func (s *StaticPodWorkload) AvailableReplicas() int32 {
	if s.Status.Phase == corev1.PodRunning {
		return 1
	}
	return 0
}

func (s *StaticPodWorkload) PodSpec() *corev1.PodSpec {
	return &s.Spec
}

// IsStaticPod return true whether the pod is static or not
// https://kubernetes.io/docs/tasks/configure-pod-container/static-pod/
func IsStaticPod(p *corev1.Pod) bool {
	var nodeOwner *metav1.OwnerReference
	for _, owner := range p.OwnerReferences {
		if owner.Kind == "Node" {
			nodeOwner = &owner
			break
		}
	}

	// static pods are owned by nodes
	if nodeOwner == nil {
		return false
	}

	// https://kubernetes.io/docs/reference/labels-annotations-taints/#kubernetes-io-config-source
	configSource, ok := p.Annotations["kubernetes.io/config.source"];
	if !ok {
		return false
	}
	return configSource == "file" || configSource == "http"
}

func PodUID(p *corev1.Pod) string {
	if IsStaticPod(p) {
		// https://kubernetes.io/docs/reference/labels-annotations-taints/#kubernetes-io-config-hash
		return p.Annotations["kubernetes.io/config.hash"]
	}

	return string(p.UID)
}

type CronJobWorkloadV1 struct {
	*batchv1.CronJob
}

type CronJobWorkloadBeta struct {
	*batchv1beta1.CronJob
}

func (c *CronJobWorkloadV1) AvailableReplicas() int32 {
	return int32(len(c.Status.Active))
}

func (c *CronJobWorkloadV1) PodSpec() *corev1.PodSpec {
	return &c.Spec.JobTemplate.Spec.Template.Spec
}

func (c *CronJobWorkloadBeta) AvailableReplicas() int32 {
	return int32(len(c.Status.Active))
}

func (c *CronJobWorkloadBeta) PodSpec() *corev1.PodSpec {
	return &c.Spec.JobTemplate.Spec.Template.Spec
}

type DeploymentConfigWorkload struct {
	*openshiftappsv1.DeploymentConfig
}

func (d *DeploymentConfigWorkload) AvailableReplicas() int32 {
	return d.Status.AvailableReplicas
}

func (d *DeploymentConfigWorkload) PodSpec() *corev1.PodSpec {
	return &d.Spec.Template.Spec
}

type ArgoRolloutWorkload struct {
	*argorolloutsv1alpha1.Rollout
}

func (d *ArgoRolloutWorkload) AvailableReplicas() int32 {
	return d.Status.AvailableReplicas
}

func (d *ArgoRolloutWorkload) PodSpec() *corev1.PodSpec {
	return &d.Spec.Template.Spec
}

func ObjectToWorkload(obj client.Object) (Workload, error) {
	switch t := obj.(type) {
	case *v1.Deployment:
		return &DeploymentWorkload{Deployment: t}, nil
	case *v1.DaemonSet:
		return &DaemonSetWorkload{DaemonSet: t}, nil
	case *v1.StatefulSet:
		return &StatefulSetWorkload{StatefulSet: t}, nil
	case *corev1.Pod:
		if IsStaticPod(t) {
			return &StaticPodWorkload{Pod: t}, nil
		}
		return nil, errors.New("currently not supporting standalone pods which are not static as workloads")
	case *batchv1.CronJob:
		return &CronJobWorkloadV1{CronJob: t}, nil
	case *batchv1beta1.CronJob:
		return &CronJobWorkloadBeta{CronJob: t}, nil
	case *openshiftappsv1.DeploymentConfig:
		return &DeploymentConfigWorkload{DeploymentConfig: t}, nil
	case *argorolloutsv1alpha1.Rollout:
		return &ArgoRolloutWorkload{Rollout: t}, nil
	default:
		return nil, errors.New("unknown kind")
	}
}
