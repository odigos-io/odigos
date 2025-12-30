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
	PodTemplateSpec() *corev1.PodTemplateSpec
	LabelSelector() *metav1.LabelSelector
}

// compile time check for interface implementation
var _ Workload = &DeploymentWorkload{}
var _ Workload = &DaemonSetWorkload{}
var _ Workload = &StatefulSetWorkload{}
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

func (d *DeploymentWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &d.Spec.Template
}

func (d *DeploymentWorkload) LabelSelector() *metav1.LabelSelector {
	return d.Spec.Selector
}

type DaemonSetWorkload struct {
	*v1.DaemonSet
}

func (d *DaemonSetWorkload) AvailableReplicas() int32 {
	return d.Status.NumberReady
}

func (d *DaemonSetWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &d.Spec.Template
}

func (d *DaemonSetWorkload) LabelSelector() *metav1.LabelSelector {
	return d.Spec.Selector
}

type StatefulSetWorkload struct {
	*v1.StatefulSet
}

func (s *StatefulSetWorkload) AvailableReplicas() int32 {
	return s.Status.ReadyReplicas
}

func (s *StatefulSetWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &s.Spec.Template
}

func (s *StatefulSetWorkload) LabelSelector() *metav1.LabelSelector {
	return s.Spec.Selector
}

type CronJobWorkloadV1 struct {
	*batchv1.CronJob
}

func (c *CronJobWorkloadV1) LabelSelector() *metav1.LabelSelector {
	return nil
}

type CronJobWorkloadBeta struct {
	*batchv1beta1.CronJob
}

func (c *CronJobWorkloadV1) AvailableReplicas() int32 {
	return int32(len(c.Status.Active))
}

func (c *CronJobWorkloadV1) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &c.Spec.JobTemplate.Spec.Template
}

func (c *CronJobWorkloadBeta) AvailableReplicas() int32 {
	return int32(len(c.Status.Active))
}

func (c *CronJobWorkloadBeta) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &c.Spec.JobTemplate.Spec.Template
}

func (c *CronJobWorkloadBeta) LabelSelector() *metav1.LabelSelector {
	return nil
}

type DeploymentConfigWorkload struct {
	*openshiftappsv1.DeploymentConfig
}

func (d *DeploymentConfigWorkload) AvailableReplicas() int32 {
	return d.Status.AvailableReplicas
}

func (d *DeploymentConfigWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return d.Spec.Template
}

func (d *DeploymentConfigWorkload) LabelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: d.Spec.Selector,
	}
}

type ArgoRolloutWorkload struct {
	*argorolloutsv1alpha1.Rollout
}

func (d *ArgoRolloutWorkload) AvailableReplicas() int32 {
	return d.Status.AvailableReplicas
}

func (d *ArgoRolloutWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &d.Spec.Template
}

func (d *ArgoRolloutWorkload) LabelSelector() *metav1.LabelSelector {
	return d.Spec.Selector
}

func ObjectToWorkload(obj client.Object) (Workload, error) {
	switch t := obj.(type) {
	case *v1.Deployment:
		return &DeploymentWorkload{Deployment: t}, nil
	case *v1.DaemonSet:
		return &DaemonSetWorkload{DaemonSet: t}, nil
	case *v1.StatefulSet:
		return &StatefulSetWorkload{StatefulSet: t}, nil
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
