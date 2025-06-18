package workload

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/common/consts"

	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Workload interface {
	client.Object
	AvailableReplicas() int32
	PodTemplateSpec() *corev1.PodTemplateSpec
}

// compile time check for interface implementation
var _ Workload = &DeploymentWorkload{}
var _ Workload = &DaemonSetWorkload{}
var _ Workload = &StatefulSetWorkload{}
var _ Workload = &CronJobWorkloadV1{}
var _ Workload = &CronJobWorkloadBeta{}

type DeploymentWorkload struct {
	*v1.Deployment
}

func (d *DeploymentWorkload) AvailableReplicas() int32 {
	return d.Status.AvailableReplicas
}

func (d *DeploymentWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &d.Spec.Template
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

type StatefulSetWorkload struct {
	*v1.StatefulSet
}

func (s *StatefulSetWorkload) AvailableReplicas() int32 {
	return s.Status.ReadyReplicas
}

func (s *StatefulSetWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &s.Spec.Template
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

func (c *CronJobWorkloadV1) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &c.Spec.JobTemplate.Spec.Template
}

func (c *CronJobWorkloadBeta) AvailableReplicas() int32 {
	return int32(len(c.Status.Active))
}

func (c *CronJobWorkloadBeta) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &c.Spec.JobTemplate.Spec.Template
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
	default:
		return nil, errors.New("unknown kind")
	}
}

// Deprecated: this should only be used for backward compatibility migration.
func IsObjectLabeledForInstrumentation(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}

	val, exists := labels[consts.OdigosInstrumentationLabel]
	if !exists {
		return false
	}

	return val == consts.InstrumentationEnabled
}

// Deprecated: this should only be used for backward compatibility migration.
func IsWorkloadInstrumentationEffectiveEnabled(ctx context.Context, kubeClient client.Client, obj client.Object) (bool, error) {
	// if the object itself is labeled, we will use that value
	workloadLabels := obj.GetLabels()
	if val, exists := workloadLabels[consts.OdigosInstrumentationLabel]; exists {
		return val == consts.InstrumentationEnabled, nil
	}

	// we will get here if the workload instrumentation label is not set.
	// no label means inherit the instrumentation value from namespace.
	var ns corev1.Namespace
	err := kubeClient.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, &ns)
	if err != nil {
		logger := log.FromContext(ctx)
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		logger.Error(err, "error fetching namespace object")
		return false, err
	}

	return IsObjectLabeledForInstrumentation(&ns), nil
}

// Deprecated: this should only be used for backward compatibility migration.
func IsInstrumentationDisabledExplicitly(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels != nil {
		val, exists := labels[consts.OdigosInstrumentationLabel]
		if exists && val == consts.InstrumentationDisabled {
			return true
		}
	}

	return false
}
