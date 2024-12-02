package instrumentation_ebpf

import (
	"context"
	"errors"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Directors     ebpf.DirectorsMap
	ConfigUpdates chan<- ebpf.ConfigUpdate
}

var (
	configUpdateTimeout    = 1 * time.Second
	errConfigUpdateTimeout = errors.New("failed to update config of workload: timeout waiting for config update")
)

func (i *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	podWorkload := workload.PodWorkload{
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
			err = director.ApplyInstrumentationConfiguration(ctx, &podWorkload, instrumentationConfig)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if i.ConfigUpdates != nil {
		// send a config update request for all the instrumentation which are part of the workload.
		// if the config request is sent, the configuration updates will occur asynchronously.
		ctx, cancel := context.WithTimeout(ctx, configUpdateTimeout)
		defer cancel()

		select {
		case i.ConfigUpdates <- ebpf.ConfigUpdate{
			PodWorkload: podWorkload,
			Config:      instrumentationConfig}:
			return ctrl.Result{}, nil
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				// returning the error to retry the reconciliation
				return ctrl.Result{}, errConfigUpdateTimeout
			}
			return ctrl.Result{}, ctx.Err()
		}
	}

	return ctrl.Result{}, nil
}
