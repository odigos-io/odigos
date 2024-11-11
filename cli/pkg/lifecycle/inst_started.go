package lifecycle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odigos-io/odigos/common"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationStarted struct {
	BaseTransition
}

func (i *InstrumentationStarted) From() State {
	return LangDetectedState
}

func (i *InstrumentationStarted) To() State {
	return InstrumentationInProgress
}

func (i *InstrumentationStarted) Execute(ctx context.Context, obj client.Object, templateSpec *corev1.PodTemplateSpec) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		i.log("Waiting for Deployment to be updated ...")
		updatedPodSpec, err := i.getPodSpecFromAPIServer(ctx, obj)
		if err != nil {
			i.log(fmt.Sprintf("Error while fetching PodSpec: %s", err.Error()))
			return false, nil
		}

		for _, c := range updatedPodSpec.Spec.Containers {
			if c.Resources.Limits != nil {
				for val := range c.Resources.Limits {
					if strings.HasPrefix(val.String(), common.OdigosResourceNamespace) {
						return true, nil
					}
				}
			}
		}
		return false, nil
	})
}

func (i *InstrumentationStarted) getPodSpecFromAPIServer(ctx context.Context, obj client.Object) (*corev1.PodTemplateSpec, error) {
	switch obj.(type) {
	case *appsv1.Deployment:
		dep, err := i.client.AppsV1().Deployments(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		return &dep.Spec.Template, nil
	case *appsv1.StatefulSet:
		ss, err := i.client.AppsV1().StatefulSets(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		return &ss.Spec.Template, nil
	case *appsv1.DaemonSet:
		ds, err := i.client.AppsV1().DaemonSets(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		return &ds.Spec.Template, nil
	}

	return nil, fmt.Errorf("unsupported object type: %T", obj)
}

var _ Transition = &InstrumentationStarted{}
