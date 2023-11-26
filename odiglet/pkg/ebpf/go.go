package ebpf

import (
	"context"
	"fmt"
	"sync"

	"github.com/keyval-dev/odigos/odiglet/pkg/kube/utils"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/consts"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"go.opentelemetry.io/auto"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"k8s.io/apimachinery/pkg/types"
)

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

	podDetailsToPids map[types.NamespacedName][]int
}

func NewInstrumentationDirectorGo() (Director, error) {
	return &InstrumentationDirectorGo{
		pidsToInstrumentation:        make(map[int]*auto.Instrumentation),
		pidsAttemptedInstrumentation: make(map[int]struct{}),
		podDetailsToPids:             make(map[types.NamespacedName][]int),
	}, nil
}

func (i *InstrumentationDirectorGo) Language() common.ProgrammingLanguage {
	return common.GoProgrammingLanguage
}

func (i *InstrumentationDirectorGo) Instrument(ctx context.Context, pid int, podDetails types.NamespacedName, podWorkload *common.PodWorkload, appName string) error {
	log.Logger.V(0).Info("Instrumenting process", "pid", pid)
	i.mux.Lock()
	defer i.mux.Unlock()
	if _, exists := i.pidsAttemptedInstrumentation[pid]; exists {
		log.Logger.V(5).Info("Process already instrumented", "pid", pid)
		return nil
	}
	i.podDetailsToPids[podDetails] = append(i.podDetailsToPids[podDetails], pid)
	i.pidsAttemptedInstrumentation[pid] = struct{}{}

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
			auto.WithResourceAttributes(utils.GetResourceAttributes(podWorkload)...),
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

func (i *InstrumentationDirectorGo) Cleanup(podDetails types.NamespacedName) {
	i.mux.Lock()
	defer i.mux.Unlock()
	pids, exists := i.podDetailsToPids[podDetails]
	if !exists {
		log.Logger.V(5).Info("No processes to cleanup for pod", "pod", podDetails)
		return
	}

	log.Logger.V(0).Info("Cleaning up ebpf go instrumentation for pod", "pod", podDetails)
	delete(i.podDetailsToPids, podDetails)

	for _, pid := range pids {
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
	for details := range i.podDetailsToPids {
		i.Cleanup(details)
	}
}

func (i *InstrumentationDirectorGo) GetInstrumentation(pid int) (*auto.Instrumentation, bool) {
	i.mux.Lock()
	defer i.mux.Unlock()
	inst, ok := i.pidsToInstrumentation[pid]
	return inst, ok
}
