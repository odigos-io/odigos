package ebpf

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/consts"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"go.opentelemetry.io/auto"
	"k8s.io/apimachinery/pkg/types"
)

type InstrumentationDirectorGo struct {
	mux                   sync.Mutex
	pidsToInstrumentation map[int]*auto.Instrumentation
	podDetailsToPids      map[types.NamespacedName][]int
}

func NewInstrumentationDirectorGo() (Director, error) {
	err := os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPPort))
	if err != nil {
		return nil, err
	}

	return &InstrumentationDirectorGo{
		pidsToInstrumentation: make(map[int]*auto.Instrumentation),
		podDetailsToPids:      make(map[types.NamespacedName][]int),
	}, nil
}

func (i *InstrumentationDirectorGo) Language() common.ProgrammingLanguage {
	return common.GoProgrammingLanguage
}

func (i *InstrumentationDirectorGo) Instrument(pid int, podDetails types.NamespacedName, appName string) error {
	log.Logger.V(0).Info("Instrumenting process", "pid", pid)
	i.mux.Lock()
	defer i.mux.Unlock()
	if _, exists := i.pidsToInstrumentation[pid]; exists {
		log.Logger.V(5).Info("Process already instrumented", "pid", pid)
		return nil
	}

	go func() {
		inst, err := auto.NewInstrumentation(auto.WithPID(pid), auto.WithServiceName(appName))
		if err != nil {
			log.Logger.Error(err, "instrumentation setup failed")
			return
		}

		i.pidsToInstrumentation[pid] = inst
		i.podDetailsToPids[podDetails] = append(i.podDetailsToPids[podDetails], pid)

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

	log.Logger.V(0).Info("Cleaning up instrumentation for pod", "pod", podDetails)
	delete(i.podDetailsToPids, podDetails)
	for _, pid := range pids {
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
