package services

import (
	"testing"

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
