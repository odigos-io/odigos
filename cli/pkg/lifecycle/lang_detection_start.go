package lifecycle

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RequestLangDetection struct {
	BaseTransition
}

var _ Transition = &RequestLangDetection{}

func (r *RequestLangDetection) From() State {
	return PreflightChecksPassed
}

func (r *RequestLangDetection) To() State {
	return LangDetectionInProgress
}

func (r *RequestLangDetection) Execute(ctx context.Context, obj client.Object, templateSpec *v1.PodTemplateSpec) error {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationEnabled
	obj.SetLabels(labels)

	switch obj.(type) {
	case *appsv1.Deployment:
		deployment := obj.(*appsv1.Deployment)
		_, err := r.client.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case *appsv1.StatefulSet:
		statefulSet := obj.(*appsv1.StatefulSet)
		_, err := r.client.AppsV1().StatefulSets(statefulSet.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
		return err
	case *appsv1.DaemonSet:
		daemonSet := obj.(*appsv1.DaemonSet)
		_, err := r.client.AppsV1().DaemonSets(daemonSet.Namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
		return err
	}

	return fmt.Errorf("unsupported object type: %T", obj)
}
