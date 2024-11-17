package ebpf

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"go.opentelemetry.io/otel/attribute"

	"k8s.io/apimachinery/pkg/types"
)

type Settings struct {
	ServiceName        string
	ResourceAttributes []attribute.KeyValue
	InitialConfig	   *odigosv1.SdkConfig
}

type Factory interface {
	CreateInstrumentation(ctx context.Context, pid int, settings Settings) (Instrumentation, error)
}

type FactoryID struct {
	Language common.ProgrammingLanguage
	OtelSdk  common.OtelSdk
}

type InstrumentationDetails struct {
	Inst              Instrumentation
	Pod               types.NamespacedName
	Lang              common.ProgrammingLanguage
	WorkloadName      string
	WorkloadNamespace string
}

type Instrumentation interface {
	Load(ctx context.Context) error
	Run(ctx context.Context) error
	Close(ctx context.Context) error
	ApplyConfig(ctx context.Context, config *odigosv1.SdkConfig) error
}
