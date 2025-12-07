package instrumentation_ebpf

import (
	"context"
	"errors"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	kubecommon "github.com/odigos-io/odigos/odiglet/pkg/kube/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	ConfigUpdates           chan<- instrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	InstrumentationRequests chan<- instrumentation.InstrumentationRequest[ebpf.K8sProcessDetails]
	DistributionGetter      *distros.Getter
}

var (
	configUpdateTimeout    = 1 * time.Second
	errConfigUpdateTimeout = errors.New("failed to update config of workload: timeout waiting for config update")
)

func (i *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	podWorkload, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
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

	// potentially send config update events to active instrumentations
	configUpdateErr := i.sendConfigUpdates(ctx, podWorkload, instrumentationConfig)
	if configUpdateErr != nil {
		return ctrl.Result{}, configUpdateErr
	}

	// potentially send instrumentation requests for processes that are part of the instrumented workload and support
	// instrumentation without restart
	instrumentationRequestErr := i.sendInstrumentationRequest(ctx, podWorkload, instrumentationConfig)
	if instrumentationRequestErr != nil {
		return ctrl.Result{}, instrumentationRequestErr
	}

	return ctrl.Result{}, nil
}

func (i *InstrumentationConfigReconciler) sendConfigUpdates(ctx context.Context, podWorkload k8sconsts.PodWorkload, instrumentationConfig *odigosv1.InstrumentationConfig) error {
	if i.ConfigUpdates == nil {
		return nil
	}

	if len(instrumentationConfig.Spec.SdkConfigs) == 0 {
		return nil
	}

	// send a config update request for all the instrumentation which are part of the workload.
	// if the config request is sent, the configuration updates will occur asynchronously.
	ctx, cancel := context.WithTimeout(ctx, configUpdateTimeout)
	defer cancel()

	configUpdate := instrumentation.ConfigUpdate[ebpf.K8sConfigGroup]{}
	for _, sdkConfig := range instrumentationConfig.Spec.SdkConfigs {
		cg := ebpf.K8sConfigGroup{Pw: podWorkload, Lang: sdkConfig.Language}
		currentConfig := sdkConfig
		configUpdate[cg] = &currentConfig
	}

	select {
	case i.ConfigUpdates <- configUpdate:
		return nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			// returning the error to retry the reconciliation
			return errConfigUpdateTimeout
		}
		return ctx.Err()
	}
}

func (i *InstrumentationConfigReconciler) sendInstrumentationRequest(ctx context.Context, podWorkload k8sconsts.PodWorkload, instrumentationConfig *odigosv1.InstrumentationConfig) error {
	// check for distributions that support instrumentation without a restart
	instrumentableContainers := make(map[string]*distro.OtelDistro)
	for _, containerConfig := range instrumentationConfig.Spec.Containers {
		d := i.DistributionGetter.GetDistroByName(containerConfig.OtelDistroName)
		if d != nil && d.RuntimeAgent != nil && d.RuntimeAgent.NoRestartRequired {
			instrumentableContainers[containerConfig.ContainerName] = d
		}
	}

	// if none of the containers support instrumentation without a restart - nothing to do here
	if len(instrumentableContainers) == 0 {
		return nil
	}

	selectedPods, err := kubecommon.WorkloadPodsOnCurrentNode(i.Client, ctx, instrumentationConfig)
	if err != nil {
		return err
	}

	if len(selectedPods) == 0 {
		return nil
	}

	// go over the (pod, container) pairs that are relevant for instrumentation
	// for each pair, find all processes that are part of that container - and send an instrumentation request
	

	return nil
}
