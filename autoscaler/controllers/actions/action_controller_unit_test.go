package actions

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestConvertActionToProcessorRejectsUnsupportedExtractAttributeSignal(t *testing.T) {
	action := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			Signals: []common.ObservabilitySignal{
				common.TracesObservabilitySignal,
				common.LogsObservabilitySignal,
			},
			ExtractAttribute: &odigosactions.ExtractAttributeConfig{
				Extractions: []odigosactions.Extraction{
					{
						TargetAttributeName: "user.id",
						LookupKey:           "user_id",
						DataFormat:          odigosactions.FormatJSON,
					},
				},
			},
		},
	}

	processor, err := convertActionToProcessor(context.Background(), nil, action)

	require.Nil(t, processor)
	require.ErrorContains(t, err, "unsupported signal in ExtractAttribute action: LOGS")
}
