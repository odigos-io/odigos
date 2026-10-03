/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	actionscatalog "github.com/odigos-io/odigos/actions"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/config"
)

// The action catalog is loaded once per process by the autoscaler binary's main; do the same here
// since signal resolution reads it.
func TestMain(m *testing.M) {
	if err := actionscatalog.Load(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func extractAttributeAction(signals ...common.ObservabilitySignal) *odigosv1.Action {
	return &odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: "extract-order-id", Namespace: "odigos-system"},
		Spec: odigosv1.ActionSpec{
			ActionName: "extract order id",
			Signals:    signals,
			ExtractAttribute: &odigosactions.ExtractAttributeConfig{
				ExtractAttributeConfig: actionsapi.ExtractAttributeConfig{
					Extractions: []actionsapi.Extraction{
						{
							TargetAttributeName: "order.id",
							LookupKey:           "order_id",
							DataFormat:          actionsapi.FormatJSON,
						},
					},
				},
			},
		},
	}
}

// The odigosextractattribute processor is traces-only. A signal the processor cannot consume must
// never reach the generated Processor: the collector fails to build that pipeline with "telemetry
// type is not supported" and crash-loops, taking every destination down with it.
func TestConvertActionToProcessor_DropsUnsupportedSignals(t *testing.T) {
	processor, err := convertActionToProcessor(context.Background(), nil, extractAttributeAction(
		common.TracesObservabilitySignal,
		common.LogsObservabilitySignal,
		common.MetricsObservabilitySignal,
	))
	require.NoError(t, err)

	assert.Equal(t, []common.ObservabilitySignal{common.TracesObservabilitySignal}, processor.Spec.Signals)
}

// End of the chain the autoscaler owns: the generated Processor must not be placed in the
// collector's logs, metrics or profiles pipelines.
func TestConvertActionToProcessor_NotWiredIntoUnsupportedPipelines(t *testing.T) {
	processor, err := convertActionToProcessor(context.Background(), nil, extractAttributeAction(
		common.TracesObservabilitySignal,
		common.LogsObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.ProfilesObservabilitySignal,
	))
	require.NoError(t, err)

	result := config.CrdProcessorToConfig([]config.ProcessorConfigurer{*processor})

	assert.NotEmpty(t, result.TracesProcessors)
	assert.Empty(t, result.LogsProcessors)
	assert.Empty(t, result.MetricsProcessors)
	assert.Empty(t, result.ProfilesProcessors)
}

// An action asking only for signals its processor cannot consume ends up in no pipeline at all,
// rather than in one that cannot instantiate it.
func TestConvertActionToProcessor_NoSupportedSignalRequested(t *testing.T) {
	processor, err := convertActionToProcessor(context.Background(), nil, extractAttributeAction(
		common.LogsObservabilitySignal,
	))
	require.NoError(t, err)

	assert.Empty(t, processor.Spec.Signals)

	result := config.CrdProcessorToConfig([]config.ProcessorConfigurer{*processor})
	assert.Empty(t, result.TracesProcessors)
	assert.Empty(t, result.LogsProcessors)
	assert.Empty(t, result.MetricsProcessors)
	assert.Empty(t, result.ProfilesProcessors)
}

// Actions whose processor handles every signal must keep the signals the user asked for.
func TestConvertActionToProcessor_KeepsSignalsSupportedByTheAction(t *testing.T) {
	action := &odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: "delete-attrs", Namespace: "odigos-system"},
		Spec: odigosv1.ActionSpec{
			ActionName: "delete attributes",
			Signals: []common.ObservabilitySignal{
				common.TracesObservabilitySignal,
				common.LogsObservabilitySignal,
				common.MetricsObservabilitySignal,
			},
			DeleteAttribute: &actionsv1.DeleteAttributeConfig{
				AttributeNamesToDelete: []string{"http.user_agent"},
			},
		},
	}

	processor, err := convertActionToProcessor(context.Background(), nil, action)
	require.NoError(t, err)

	assert.ElementsMatch(t, action.Spec.Signals, processor.Spec.Signals)
}
