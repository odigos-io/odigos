package instrumentation_ebpf

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Directors ebpf.DirectorsMap
	OnUpdate  ebpf.ConfigUpdateFunc
}

func (i *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	podWorkload := &workload.PodWorkload{
		Namespace: req.Namespace,
		Kind:      workloadKind,
		Name:      workloadName,
	}

	// Fetch the InstrumentationConfig instrumentationConfig
	instrumentationConfig := &odigosv1.InstrumentationConfig{}
	err = i.Get(ctx, req.NamespacedName, instrumentationConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	langs := instrumentationConfig.Languages()

	for key, director := range i.Directors {
		// Apply the configuration only for languages specified in the InstrumentationConfig
		if _, ok := langs[key.Language]; ok {
			err = director.ApplyInstrumentationConfiguration(ctx, podWorkload, instrumentationConfig)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if i.OnUpdate != nil {
		err = i.OnUpdate(ctx, types.NamespacedName{Namespace: req.Namespace, Name: workloadName}, instrumentationConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
