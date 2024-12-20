package startlangdetection

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (s *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var source v1alpha1.Source
	err := s.Get(ctx, req.NamespacedName, &source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(req.Name, source.Spec.Workload.Kind)
	obj := workload.ClientObjectFromWorkloadKind(source.Spec.Workload.Kind)

	err = requestOdigletsToCalculateRuntimeDetails(ctx, s.Client, instConfigName, req.Namespace, obj, s.Scheme)
	return ctrl.Result{}, err
}
