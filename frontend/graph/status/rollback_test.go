package status

import (
	"testing"

	"github.com/stretchr/testify/require"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestCalculateAutoRollbackStatus_NilConfigDoesNotPanic(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = true

	status := CalculateAutoRollbackStatus(ic, nil)

	require.NotNil(t, status)
	require.Equal(t, model.DesiredStateProgressFailure, status.Status)
	require.NotNil(t, status.ReasonEnum)
	require.Equal(t, string(AutoRollbackReasonInvalidConfig), *status.ReasonEnum)
}
