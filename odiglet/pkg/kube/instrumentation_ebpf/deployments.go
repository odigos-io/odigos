package instrumentation_ebpf

import (
	"context"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DeploymentsReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Directors map[common.ProgrammingLanguage]ebpf.Director
}

func (d *DeploymentsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconciling deployment", "name", request.Name, "namespace", request.Namespace)
	err := ApplyEbpfToPodWorkload(ctx, d.Client, d.Directors, &PodWorkload{
		Name:      request.Name,
		Namespace: request.Namespace,
		Kind:      "Deployment",
	})

	return ctrl.Result{}, err
}
