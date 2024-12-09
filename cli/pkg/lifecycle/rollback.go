package lifecycle

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/odigos-io/odigos/common/consts"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (o *Orchestrator) rollBack(obj client.Object, templateSpecFetcher PodTemplateSpecFetcher) error {
	// We create a new context for the rollback operation to ensure that the operation is not cancelled by the parent context
	ctx := context.Background()

	o.log("Rolling back changes to deployment")
	templateSpec, err := templateSpecFetcher(ctx, obj.GetName(), obj.GetNamespace())
	if err != nil {
		o.log("Error fetching template spec")
		return err
	}

	if isObjectModifiedByOdigos(obj, templateSpec) {
		err := patchOdigosLabel(ctx, o.Client, obj)
		if err != nil {
			return err
		}

		err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Minute, true, func(ctx context.Context) (bool, error) {
			templateSpec, err := templateSpecFetcher(ctx, obj.GetName(), obj.GetNamespace())
			if err != nil {
				o.log("Error fetching template spec")
				return false, err
			}

			for _, container := range templateSpec.Spec.Containers {
				if workload.IsContainerInstrumented(&container) {
					return false, nil
				}
			}

			rolloutCompleted, err := utils.VerifyAllPodsAreNOTInstrumented(ctx, o.Client, obj)
			if err != nil {
				o.log("Error verifying all pods are not instrumented")
				return false, err
			}

			if rolloutCompleted {
				o.log("Rollout completed, all running pods does not contains instrumentation")
			}

			return rolloutCompleted, nil
		})

	} else {
		o.log("No changes made by Odigos, skipping rollback")
	}
	return nil
}

func patchOdigosLabel(ctx context.Context, client kubernetes.Interface, obj client.Object) error {
	labels := obj.GetLabels()
	if labels != nil {
		if _, ok := labels[consts.OdigosInstrumentationLabel]; !ok {
			return nil
		}
	}
	patch := fmt.Sprintf(`{"metadata":{"labels":{"%s":null}}}`, consts.OdigosInstrumentationLabel)

	switch obj.(type) {
	case *appsv1.Deployment:
		_, err := client.AppsV1().Deployments(obj.GetNamespace()).Patch(
			ctx,
			obj.GetName(),
			types.MergePatchType,
			[]byte(patch),
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	case *appsv1.StatefulSet:
		_, err := client.AppsV1().StatefulSets(obj.GetNamespace()).Patch(
			ctx,
			obj.GetName(),
			types.MergePatchType,
			[]byte(patch),
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	case *appsv1.DaemonSet:
		_, err := client.AppsV1().DaemonSets(obj.GetNamespace()).Patch(
			ctx,
			obj.GetName(),
			types.MergePatchType,
			[]byte(patch),
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	}
	return nil

}

func isObjectModifiedByOdigos(obj client.Object, templateSpec *v1.PodTemplateSpec) bool {
	if workload.IsObjectLabeledForInstrumentation(obj) {
		return true
	}

	for _, container := range templateSpec.Spec.Containers {
		if workload.IsContainerInstrumented(&container) {
			return true
		}
	}

	return false
}
