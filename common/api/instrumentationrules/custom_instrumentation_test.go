package instrumentationrules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomInstrumentationsVerifyValidatesCpp(t *testing.T) {
	err := (&CustomInstrumentations{
		Cpp: []CppCustomProbe{{}},
	}).Verify()

	require.ErrorContains(t, err, "invalid configuration for cpp custom instrumentation")
	require.ErrorContains(t, err, "signature is required")
}
