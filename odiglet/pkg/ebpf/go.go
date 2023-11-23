package ebpf

import (
	"context"
	"fmt"
	"sync"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/consts"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"go.opentelemetry.io/auto"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"k8s.io/apimachinery/pkg/types"
)

type podDetails struct {
	Workload common.PodWorkload
	Pids     []int
}

type InstrumentationDirectorGo struct {
	mux sync.Mutex

	// this map holds the instrumentation object which is used to close the instrumentation
	// the map is filled only after the instrumentation is actually created
	// which is an asyn process that might take some time
	pidsToInstrumentation map[int]*auto.Instrumentation

	// this map is used to make sure we do not attempt to instrument the same process twice.
	// it keeps track of which processes we already attempted to instrument,
	// so we can avoid attempting to instrument them again.
	pidsAttemptedInstrumentation map[int]struct{}

	// via this map, we can find the workload and pids for a specific pod.
	// sometimes we only have the pod name and namespace, so this map is useful.
	podsToDetails map[types.NamespacedName]podDetails

	// this map can be used when we only have the workload, and need to find the pods to derive pids.
	workloadToPods map[common.PodWorkload]map[types.NamespacedName]struct{}
}

func NewInstrumentationDirectorGo() (Director, error) {
	return &InstrumentationDirectorGo{
		pidsToInstrumentation:        make(map[int]*auto.Instrumentation),
		pidsAttemptedInstrumentation: make(map[int]struct{}),
		podsToDetails:                make(map[types.NamespacedName]podDetails),
		workloadToPods:               make(map[common.PodWorkload]map[types.NamespacedName]struct{}),
	}, nil
}

func (i *InstrumentationDirectorGo) Language() common.ProgrammingLanguage {
	return common.GoProgrammingLanguage
}

func (i *InstrumentationDirectorGo) Instrument(ctx context.Context, pid int, pod types.NamespacedName, podWorkload common.PodWorkload, appName string) error {
	log.Logger.V(0).Info("Instrumenting process", "pid", pid)
	i.mux.Lock()
	defer i.mux.Unlock()
	if _, exists := i.pidsAttemptedInstrumentation[pid]; exists {
		log.Logger.V(5).Info("Process already instrumented", "pid", pid)
		return nil
	}

	details, exists := i.podsToDetails[pod]
	if !exists {
		details = podDetails{
			Workload: podWorkload,
			Pids:     []int{},
		}
		i.podsToDetails[pod] = details
	}
	details.Pids = append(details.Pids, pid)
	i.pidsAttemptedInstrumentation[pid] = struct{}{}
	if _, exists := i.workloadToPods[podWorkload]; !exists {
		i.workloadToPods[podWorkload] = make(map[types.NamespacedName]struct{})
	}
	i.workloadToPods[podWorkload][pod] = struct{}{}

	defaultExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", env.Current.NodeIP, consts.OTLPPort)),
	)
	if err != nil {
		log.Logger.Error(err, "failed to create exporter")
		return err
	}

	go func() {
		inst, err := auto.NewInstrumentation(
			ctx,
			auto.WithPID(pid),
			auto.WithServiceName(appName),
			auto.WithTraceExporter(defaultExporter),
		)
		if err != nil {
			log.Logger.Error(err, "instrumentation setup failed")
			return
		}

		i.mux.Lock()
		_, stillExists := i.pidsAttemptedInstrumentation[pid]
		if stillExists {
			i.pidsToInstrumentation[pid] = inst
			i.mux.Unlock()
		} else {
			i.mux.Unlock()
			// we attempted to instrument this process, but it was already cleaned up
			// so we need to clean up the instrumentation we just created
			err = inst.Close()
			if err != nil {
				log.Logger.Error(err, "error cleaning up instrumentation for process", "pid", pid)
			}
			return
		}

		if err := inst.Run(context.Background()); err != nil {
			log.Logger.Error(err, "instrumentation crashed after running")
		}
	}()

	return nil
}

func (i *InstrumentationDirectorGo) Cleanup(pod types.NamespacedName) {
	i.mux.Lock()
	defer i.mux.Unlock()
	details, exists := i.podsToDetails[pod]
	if !exists {
		log.Logger.V(5).Info("No processes to cleanup for pod", "pod", pod)
		return
	}

	log.Logger.V(0).Info("Cleaning up ebpf go instrumentation for pod", "pod", pod)
	delete(i.podsToDetails, pod)

	// clear the pod from the workloadToPods map
	workload := details.Workload
	delete(i.workloadToPods[workload], pod)
	if len(i.workloadToPods[workload]) == 0 {
		delete(i.workloadToPods, workload)
	}

	for _, pid := range details.Pids {
		delete(i.pidsAttemptedInstrumentation, pid)

		inst, exists := i.pidsToInstrumentation[pid]
		if !exists {
			log.Logger.V(5).Info("No objects to cleanup for process", "pid", pid)
			continue
		}

		delete(i.pidsToInstrumentation, pid)
		go func() {
			err := inst.Close()
			if err != nil {
				log.Logger.Error(err, "error cleaning up objects for process", "pid", pid)
			}
		}()
	}
}

func (i *InstrumentationDirectorGo) Shutdown() {
	log.Logger.V(0).Info("Shutting down instrumentation director")
	for details := range i.podsToDetails {
		i.Cleanup(details)
	}
}

func (i *InstrumentationDirectorGo) GetWorkloadInstrumentations(workload common.PodWorkload) []*auto.Instrumentation {
	i.mux.Lock()
	defer i.mux.Unlock()
	pods, ok := i.workloadToPods[workload]
	if !ok {
		return nil
	}

	var insts []*auto.Instrumentation
	for pod := range pods {
		details, ok := i.podsToDetails[pod]
		if !ok {
			continue
		}

		for _, pid := range details.Pids {
			inst, ok := i.pidsToInstrumentation[pid]
			if !ok {
				continue
			}

			insts = append(insts, inst)
		}
	}

	return insts
}
