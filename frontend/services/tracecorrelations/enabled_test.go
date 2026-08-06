package tracecorrelations

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestServiceIOEnabled(t *testing.T) {
	require.False(t, ServiceIOEnabled(nil))
	require.False(t, ServiceIOEnabled(&common.OdigosConfiguration{}))
	require.False(t, ServiceIOEnabled(&common.OdigosConfiguration{
		TraceCorrelations: &common.TraceCorrelationsConfiguration{},
	}))
	require.False(t, ServiceIOEnabled(&common.OdigosConfiguration{
		TraceCorrelations: &common.TraceCorrelationsConfiguration{
			ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{},
		},
	}))
	require.False(t, ServiceIOEnabled(&common.OdigosConfiguration{
		TraceCorrelations: &common.TraceCorrelationsConfiguration{
			ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
				Enabled: boolPtr(false),
			},
		},
	}))
	require.True(t, ServiceIOEnabled(&common.OdigosConfiguration{
		TraceCorrelations: &common.TraceCorrelationsConfiguration{
			ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
				Enabled: boolPtr(true),
			},
		},
	}))
}
