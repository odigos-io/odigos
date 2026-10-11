package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestConvertPiiMaskingFromInputAllCategories(t *testing.T) {
	cfg, err := convertPiiMaskingFromInput(&model.ActionFieldsInput{
		PiiCategories: []string{"CREDIT_CARD", "EMAIL", "JWT", "UUID"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, []actionsapi.PiiCategory{
		actionsapi.CreditCardMasking,
		actionsapi.EmailMasking,
		actionsapi.JwtMasking,
		actionsapi.UuidMasking,
	}, cfg.PiiCategories)
}

func TestConvertPiiMaskingFromInputRejectsUnsupportedCategory(t *testing.T) {
	cfg, err := convertPiiMaskingFromInput(&model.ActionFieldsInput{
		PiiCategories: []string{"CREDIT_CARD", "PHONE"},
	}, nil)

	require.Nil(t, cfg)
	require.ErrorContains(t, err, `unsupported pii category "PHONE"`)
}

func TestGetSpecFromInputPiiMaskingAllCategories(t *testing.T) {
	spec, err := getSpecFromInput(model.ActionInput{
		Type:     model.ActionTypePiiMasking,
		Disabled: false,
		Signals:  []model.SignalType{model.SignalTypeTraces},
		Fields: &model.ActionFieldsInput{
			PiiCategories: []string{"CREDIT_CARD", "EMAIL", "JWT", "UUID"},
		},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, spec.PiiMasking)
	require.Equal(t, []actionsapi.PiiCategory{
		actionsapi.CreditCardMasking,
		actionsapi.EmailMasking,
		actionsapi.JwtMasking,
		actionsapi.UuidMasking,
	}, spec.PiiMasking.PiiCategories)
}

func TestConvertActionToModelPiiMaskingScopes(t *testing.T) {
	action := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			PiiMasking: &odigosactions.PiiMaskingConfig{
				Scopes: &k8sconsts.SourcesScopes{Namespaces: []string{"default"}},
				PiiMaskingConfig: actionsapi.PiiMaskingConfig{
					PiiCategories: []actionsapi.PiiCategory{actionsapi.EmailMasking},
				},
			},
		},
	}

	out, err := convertActionToModel(action)

	require.NoError(t, err)
	require.Equal(t, model.ActionTypePiiMasking, out.Type)
	require.NotNil(t, out.Fields.Scopes)
	require.Equal(t, []string{"default"}, out.Fields.Scopes.Namespaces)
}

func TestConvertActionToModelPiiMaskingWithoutScopes(t *testing.T) {
	action := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			PiiMasking: &odigosactions.PiiMaskingConfig{
				PiiMaskingConfig: actionsapi.PiiMaskingConfig{
					PiiCategories: []actionsapi.PiiCategory{actionsapi.EmailMasking},
				},
			},
		},
	}

	out, err := convertActionToModel(action)

	require.NoError(t, err)
	require.Nil(t, out.Fields.Scopes)
}
