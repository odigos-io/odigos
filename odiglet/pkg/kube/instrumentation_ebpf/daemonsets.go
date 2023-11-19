package instrumentation_ebpf

import (
	"context"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DaemonSetsReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Directors map[common.ProgrammingLanguage]ebpf.Director
}

func (d *DaemonSetsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	err := ApplyEbpfToPodWorkload(ctx, d.Client, d.Directors, &PodWorkload{
		Name:      request.Name,
		Namespace: request.Namespace,
		Kind:      "DaemonSet",
	})

	return ctrl.Result{}, err
}
