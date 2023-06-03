package ebpf

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/keyval-dev/odigos/odiglet/pkg/env"

	"github.com/keyval-dev/odigos/common/consts"

	"go.opentelemetry.io/auto/pkg/instrumentors"

	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	log2 "go.opentelemetry.io/auto/pkg/log"
	"go.opentelemetry.io/auto/pkg/process"
	"k8s.io/apimachinery/pkg/types"
)

type Director interface {
	Instrument(pid int, podDetails types.NamespacedName, appName string) error
	Cleanup(podDetails types.NamespacedName)
	Shutdown()
}

type InstrumentationDirector struct {
	analyzer         *process.Analyzer
	relevantFuncs    map[string]interface{}
	mux              sync.Mutex
	pidsToObjects    map[int]*Objects
	podDetailsToPids map[types.NamespacedName][]int
}

func NewInstrumentationDirector() (*InstrumentationDirector, error) {
	// TODO: less hacky after OpenTelemetry go changes
	log2.Init()
	err := os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPPort))
	if err != nil {
		return nil, err
	}

	mgr, err := instrumentors.NewManager(nil)
	if err != nil {
		return nil, err
	}

	return &InstrumentationDirector{
		analyzer:         &process.Analyzer{},
		relevantFuncs:    mgr.GetRelevantFuncs(),
		pidsToObjects:    make(map[int]*Objects),
		podDetailsToPids: make(map[types.NamespacedName][]int),
	}, nil
}

func (i *InstrumentationDirector) Instrument(pid int, podDetails types.NamespacedName, appName string) error {
	log.Logger.V(0).Info("Instrumenting process", "pid", pid)
	details, err := i.analyzer.Analyze(pid, i.relevantFuncs)
	if err != nil {
		if errors.Is(err, process.ErrNotGoExe) {
			log.Logger.V(5).Info("Process is not a Go executable", "pid", pid)
			return nil
		}

		return err
	}

	log.Logger.V(0).Info("Instrumentation details", "details", details)
	i.instrumentTarget(details, podDetails, appName)
	return nil
}

func (i *InstrumentationDirector) instrumentTarget(target *process.TargetDetails, podDetails types.NamespacedName, appName string) {
	i.mux.Lock()
	defer i.mux.Unlock()
	if _, exists := i.pidsToObjects[target.PID]; exists {
		log.Logger.V(5).Info("Process already instrumented", "pid", target.PID)
		return
	}

	objects := NewObjects(target, appName)
	i.pidsToObjects[target.PID] = objects
	i.podDetailsToPids[podDetails] = append(i.podDetailsToPids[podDetails], target.PID)
	go func() {
		err := objects.Init()
		if err != nil {
			log.Logger.Error(err, "error initializing objects for process", "pid", target.PID)
			i.Cleanup(podDetails)
			return
		}
	}()
}

func (i *InstrumentationDirector) Cleanup(podDetails types.NamespacedName) {
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
		objs, exists := i.pidsToObjects[pid]
		if !exists {
			log.Logger.V(5).Info("No objects to cleanup for process", "pid", pid)
			continue
		}

		delete(i.pidsToObjects, pid)
		err := objs.Cleanup()
		if err != nil {
			log.Logger.Error(err, "error cleaning up objects for process", "pid", pid)
		}
	}
}

func (i *InstrumentationDirector) Shutdown() {
	log.Logger.V(0).Info("Shutting down instrumentation director")
	for details := range i.podDetailsToPids {
		i.Cleanup(details)
	}
}
