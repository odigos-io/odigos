package services

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestApplyTraceCorrelationsInput(t *testing.T) {
	enabled := true
	interval := "30s"
	input := &model.LocalUIConfigTraceCorrelationsInput{
		ServiceIo: &model.LocalUIConfigTraceCorrelationsServiceIOInput{
			Enabled:               &enabled,
			InputSpanAttributes:   []string{"http.route", "rpc.service"},
			OutputSpanAttributes:  []string{"db.system"},
			MetricsFlushInterval:  &interval,
		},
	}

	cfg := &common.OdigosConfiguration{}
	applyTraceCorrelationsInput(cfg, input)

	require.NotNil(t, cfg.TraceCorrelations)
	require.NotNil(t, cfg.TraceCorrelations.ServiceIO)
	require.True(t, *cfg.TraceCorrelations.ServiceIO.Enabled)
	require.Equal(t, []string{"http.route", "rpc.service"}, cfg.TraceCorrelations.ServiceIO.InputSpanAttributes)
	require.Equal(t, []string{"db.system"}, cfg.TraceCorrelations.ServiceIO.OutputSpanAttributes)
	require.Equal(t, "30s", cfg.TraceCorrelations.ServiceIO.MetricsFlushInterval)
}
