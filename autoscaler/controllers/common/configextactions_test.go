package common

import (
	"os"
	"testing"

	actionscatalog "github.com/odigos-io/odigos/actions"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The action catalog is loaded once per process by each binary's main; do the same here since
// config-extension resolution reads it.
func TestMain(m *testing.M) {
	if err := actionscatalog.Load(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func piiMaskingAction(name string, signals ...odigoscommon.ObservabilitySignal) odigosv1.Action {
	return odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system"},
		Spec: odigosv1.ActionSpec{
			ActionName: name,
			Signals:    signals,
			PiiMasking: &odigosactions.PiiMaskingConfig{},
		},
	}
}

func processorByType(t *testing.T, processors []odigosv1.Processor, processorType string) *odigosv1.Processor {
	t.Helper()
	for i := range processors {
		if processors[i].Spec.Type == processorType {
			return &processors[i]
		}
	}
	return nil
}

// The odigospiimasking processor is traces-only. A signal the processor cannot consume must never
// reach the collector config: the collector fails to build the pipeline with "telemetry type is not
// supported" and crash-loops, taking every destination down with it.
func TestConvertActionsToConfigExtensionProcessors_DropsUnsupportedSignals(t *testing.T) {
	processors := ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{
			piiMaskingAction("mask-all",
				odigoscommon.TracesObservabilitySignal,
				odigoscommon.LogsObservabilitySignal,
				odigoscommon.MetricsObservabilitySignal,
			),
		},
	})

	proc := processorByType(t, processors, consts.OdigosPiiMaskingProcessorType)
	require.NotNil(t, proc, "expected a %s processor", consts.OdigosPiiMaskingProcessorType)
	assert.ElementsMatch(t, []odigoscommon.ObservabilitySignal{odigoscommon.TracesObservabilitySignal}, proc.Spec.Signals)
}

// An action that asks only for signals its processor cannot consume produces no processor at all,
// rather than one wired into a pipeline that cannot instantiate it.
func TestConvertActionsToConfigExtensionProcessors_NoProcessorWhenNoSupportedSignal(t *testing.T) {
	processors := ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{
			piiMaskingAction("mask-logs", odigoscommon.LogsObservabilitySignal),
		},
	})

	assert.Nil(t, processorByType(t, processors, consts.OdigosPiiMaskingProcessorType))
}

// End of the chain the autoscaler owns: the generated Processor must not place the traces-only
// processor in the collector's logs or metrics pipelines.
func TestConvertActionsToConfigExtensionProcessors_NotWiredIntoNonTracesPipelines(t *testing.T) {
	processors := ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{
			piiMaskingAction("mask-all",
				odigoscommon.TracesObservabilitySignal,
				odigoscommon.LogsObservabilitySignal,
				odigoscommon.MetricsObservabilitySignal,
			),
		},
	})

	list := odigosv1.ProcessorList{Items: processors}
	configurers := ToProcessorConfigurerArray(FilterAndSortProcessorsByOrderHint(&list, odigosv1.CollectorsGroupRoleClusterGateway))
	result := config.CrdProcessorToConfig(configurers)

	assert.Empty(t, result.LogsProcessors)
	assert.Empty(t, result.MetricsProcessors)
	assert.Empty(t, result.ProfilesProcessors)
	assert.NotEmpty(t, result.TracesProcessors)
}
