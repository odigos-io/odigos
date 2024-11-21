package instrumentationconfig

import (
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkloadsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *WorkloadsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var annotations map[string]string

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, deployment); err == nil {
		annotations = deployment.Annotations
	} else {
		// Check for StatefulSet
		statefulset := &appsv1.StatefulSet{}
		if err := r.Get(ctx, req.NamespacedName, statefulset); err == nil {
			annotations = statefulset.Annotations
		} else {
			// Check for DaemonSet
			daemonset := &appsv1.DaemonSet{}
			if err := r.Get(ctx, req.NamespacedName, daemonset); err == nil {
				annotations = daemonset.Annotations
			} else {
				// None of the workloads matched; ignore the request
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
	}

	reportedName := annotations[consts.OdigosReportedNameAnnotation]

	var instrumentationConfig odigosv1alpha1.InstrumentationConfig
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, &instrumentationConfig); err != nil {
		if errors.IsNotFound(err) {
			// Create a new InstrumentationConfig if it doesn't exist
			instrumentationConfig = odigosv1alpha1.InstrumentationConfig{
				ObjectMeta: v1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
				Spec: odigosv1alpha1.InstrumentationConfigSpec{
					ServiceName: reportedName,
				},
			}
			if err := r.Create(ctx, &instrumentationConfig); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Update the InstrumentationConfig if the serviceName has changed
	if instrumentationConfig.Spec.ServiceName != reportedName {
		instrumentationConfig.Spec.ServiceName = reportedName
		if err := r.Update(ctx, &instrumentationConfig); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
