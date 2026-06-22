package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOdigosLogLevelCompareOrdersByVerbosity(t *testing.T) {
	require.Positive(t, LogLevelDebug.Compare(LogLevelInfo))
	require.Positive(t, LogLevelInfo.Compare(LogLevelWarn))
	require.Positive(t, LogLevelWarn.Compare(LogLevelError))

	require.Negative(t, LogLevelError.Compare(LogLevelWarn))
	require.Negative(t, LogLevelWarn.Compare(LogLevelInfo))
	require.Negative(t, LogLevelInfo.Compare(LogLevelDebug))

	require.Zero(t, LogLevelDebug.Compare(LogLevelDebug))
}
