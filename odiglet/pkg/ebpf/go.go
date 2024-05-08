package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/odiglet/pkg/kube/utils"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/consts"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"go.opentelemetry.io/auto"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

type GoOtelEbpfSdk struct {
	inst *auto.Instrumentation
}

type GoInstrumentationFactory struct{}

func NewGoInstrumentationFactory() InstrumentationFactory[*GoOtelEbpfSdk] {
	return &GoInstrumentationFactory{}
}

func (g *GoInstrumentationFactory) CreateEbpfInstrumentation(ctx context.Context, pid int, serviceName string, podWorkload *common.PodWorkload, containerName string, podName string) (*GoOtelEbpfSdk, error) {
	defaultExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", env.Current.NodeIP, consts.OTLPPort)),
	)
	if err != nil {
		log.Logger.Error(err, "failed to create exporter")
		return nil, err
	}

	inst, err := auto.NewInstrumentation(
		ctx,
		auto.WithPID(pid),
		auto.WithResourceAttributes(utils.GetResourceAttributes(podWorkload, podName)...),
		auto.WithServiceName(serviceName),
		auto.WithTraceExporter(defaultExporter),
		auto.WithGlobal(),
	)
	if err != nil {
		log.Logger.Error(err, "instrumentation setup failed")
		return nil, err
	}

	return &GoOtelEbpfSdk{inst: inst}, nil
}

func (g *GoOtelEbpfSdk) Run(ctx context.Context) error {
	return g.inst.Run(ctx)
}

func (g *GoOtelEbpfSdk) Close(ctx context.Context) error {
	return g.inst.Close()
}
