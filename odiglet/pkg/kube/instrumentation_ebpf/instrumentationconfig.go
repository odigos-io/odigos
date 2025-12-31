package instrumentation_ebpf

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/instrumentation/detector"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	kubecommon "github.com/odigos-io/odigos/odiglet/pkg/kube/common"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	procdiscovery "github.com/odigos-io/odigos/procdiscovery/pkg/process"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	ConfigUpdates           chan<- instrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	InstrumentationRequests chan<- instrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails]
	DistributionGetter      *distros.Getter
}

func (i *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	podWorkload, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the InstrumentationConfig
	instrumentationConfig := &odigosv1.InstrumentationConfig{}
	err = i.Get(ctx, req.NamespacedName, instrumentationConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// if the instrumentationConfig is deleted, send un-instrumentation request for the workload
			err = i.sendUnInstrumentationRequest(podWorkload)
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, err
		}
	}

	// potentially send config update events to active instrumentations
	configUpdateErr := i.sendConfigUpdates(ctx, podWorkload, instrumentationConfig)
	if configUpdateErr != nil {
		return ctrl.Result{}, configUpdateErr
	}

	// check if there are any enabled containers in the instrumentation config
	enabledContainers := make(map[string]struct{})
	for _, containerConfig := range instrumentationConfig.Spec.Containers {
		if containerConfig.OtelDistroName == "" || !containerConfig.AgentEnabled {
			continue
		}
		enabledContainers[containerConfig.ContainerName] = struct{}{}
	}
	// if no enabled containers, send un-instrumentation request
	// note: we might miss some edge cases here: if workload has multiple containers and only some are disabled,
	// we should ideally un-instrument only the disabled ones.
	if len(enabledContainers) == 0 {
		err = i.sendUnInstrumentationRequest(podWorkload)
		return ctrl.Result{}, err
	}

	// potentially send instrumentation requests for processes that are part of the instrumented workload and support
	// instrumentation without restart
	instrumentationRequestErr := i.sendInstrumentationRequest(ctx, podWorkload, instrumentationConfig)
	if instrumentationRequestErr != nil {
		return ctrl.Result{}, instrumentationRequestErr
	}

	return ctrl.Result{}, nil
}

func (i *InstrumentationConfigReconciler) sendUnInstrumentationRequest(podWorkload k8sconsts.PodWorkload) error {
	ir := instrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails]{
		Instrument:   false,
		ProcessGroup: ebpf.K8sProcessGroup{Pw: podWorkload},
	}
	select {
	case i.InstrumentationRequests <- ir:
		return nil
	default:
		return errors.New("failed to send un-instrumentation request, consumer is busy")
	}
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
	configUpdate := instrumentation.ConfigUpdate[ebpf.K8sConfigGroup]{}
	for _, sdkConfig := range instrumentationConfig.Spec.SdkConfigs {
		cg := ebpf.K8sConfigGroup{Pw: podWorkload, Lang: sdkConfig.Language}
		currentConfig := sdkConfig
		configUpdate[cg] = &currentConfig
	}

	select {
	case i.ConfigUpdates <- configUpdate:
		return nil
	default:
		return errors.New("failed to send config update, consumer is busy")
	}
}

// sendInstrumentationRequest sends an instrumentation request for all processes that are part of the given workload
// and run in containers that support instrumentation without a restart.
func (i *InstrumentationConfigReconciler) sendInstrumentationRequest(ctx context.Context, podWorkload k8sconsts.PodWorkload, instrumentationConfig *odigosv1.InstrumentationConfig) error {
	// check for distributions that support instrumentation without a restart
	distroByContainer := make(map[string]*distro.OtelDistro)
	for _, containerConfig := range instrumentationConfig.Spec.Containers {
		if containerConfig.OtelDistroName == "" || !containerConfig.AgentEnabled {
			continue
		}
		d := i.DistributionGetter.GetDistroByName(containerConfig.OtelDistroName)
		if d != nil && d.RuntimeAgent != nil && d.RuntimeAgent.NoRestartRequired {
			distroByContainer[containerConfig.ContainerName] = d
		}
	}

	// if none of the containers support instrumentation without a restart, nothing to do here
	if len(distroByContainer) == 0 {
		return nil
	}

	selectedPods, err := kubecommon.WorkloadPodsOnCurrentNode(i.Client, ctx, instrumentationConfig)
	if err != nil {
		return err
	}

	if len(selectedPods) == 0 {
		return nil
	}

	// build all the relevant (podUID, containerName) combinations for this node
	pcs := make([]process.PodContainerUID, 0, len(selectedPods)*len(distroByContainer))
	for _, p := range selectedPods {
		for c := range distroByContainer {
			pcs = append(pcs, process.PodContainerUID{
				PodUID:        string(p.UID),
				ContainerName: c,
			})
		}
	}

	// group relevant processes by (podUID, containerName)
	// this is an expensive operation and can be optimized in the future
	pidsByPodContainer, err := process.GroupByPodContainer(pcs)
	if err != nil {
		return err
	}
	if len(pidsByPodContainer) == 0 {
		return nil
	}

	// build the instrumentation request including all the relevant processes
	// for the workload on this node
	ir := instrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails]{
		Instrument:          true,
		ProcessDetailsByPid: make(map[int]*ebpf.K8sProcessDetails, len(pidsByPodContainer)),
	}
	podByUID := make(map[string]*corev1.Pod, len(selectedPods))
	for _, p := range selectedPods {
		podByUID[string(p.UID)] = &p
	}
	for podContainer, pidSet := range pidsByPodContainer {
		distribution, ok := distroByContainer[podContainer.ContainerName]
		if !ok {
			continue
		}
		for pid := range pidSet {
			details := procdiscovery.GetPidDetails(pid, nil)
			ir.ProcessDetailsByPid[pid] = &ebpf.K8sProcessDetails{
				ContainerName: podContainer.ContainerName,
				Distro:        distribution,
				Pw:            &podWorkload,
				Pod:           podByUID[podContainer.PodUID],
				ProcEvent: detector.ProcessEvent{
					EventType: detector.ProcessExecEvent,
					PID:       pid,
					ExecDetails: &detector.ExecDetails{
						ExePath:      details.ExePath,
						CmdLine:      details.CmdLine,
						Environments: details.Environments.DetailedEnvs,
					},
				},
			}
		}
	}

	// try to send the request, return an error if the consumer is busy
	// the caller (controller) can retry/requeue the handling
	select {
	case i.InstrumentationRequests <- ir:
		return nil
	default:
		return errors.New("failed to send instrumentation request, consumer is busy")
	}
}
