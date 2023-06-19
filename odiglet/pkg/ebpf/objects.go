package ebpf

import (
	"go.opentelemetry.io/auto/pkg/instrumentors"
	"go.opentelemetry.io/auto/pkg/opentelemetry"
	"go.opentelemetry.io/auto/pkg/process"
)

// Objects is a collection of objects that are needed for the eBPF instrumentation per process.
type Objects struct {
	AppName               string
	TargetDetails         *process.TargetDetails
	Controller            *opentelemetry.Controller
	InstrumentationManger *instrumentors.Manager
}

func NewObjects(details *process.TargetDetails, appName string) *Objects {
	return &Objects{
		TargetDetails: details,
		AppName:       appName,
	}
}

func (o *Objects) Init() error {
	otelController, err := opentelemetry.NewControllerWithServiceName(o.AppName)
	if err != nil {
		return err
	}

	o.Controller = otelController
	instManager, err := instrumentors.NewManager(otelController)
	if err != nil {
		return err
	}

	o.InstrumentationManger = instManager
	analyzer := &process.Analyzer{}
	allocDetails, err := analyzer.AllocateMemory(o.TargetDetails)
	if err != nil {
		return err
	}

	o.TargetDetails.AllocationDetails = allocDetails
	o.InstrumentationManger.FilterUnusedInstrumentors(o.TargetDetails)

	return o.InstrumentationManger.Run(o.TargetDetails)
}

func (o *Objects) Cleanup() error {
	o.Controller.Close()
	o.InstrumentationManger.Close()
	return nil
}
