package mockdestinationexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestConfigValidateEncoding(t *testing.T) {
	for _, encoding := range []EncodingType{EncodingNone, EncodingProto, EncodingJSON} {
		assert.NoError(t, (&Config{Encoding: encoding}).Validate())
	}
	assert.Error(t, (&Config{Encoding: "yaml"}).Validate())
}

func TestExportWithEncoding(t *testing.T) {
	for _, encoding := range []EncodingType{EncodingNone, EncodingProto, EncodingJSON} {
		exporter, err := NewMockDestinationExporter(
			&Config{Encoding: encoding},
			exportertest.NewNopSettings(exportertest.NopType),
		)
		require.NoError(t, err)

		assert.NoError(t, exporter.ConsumeTraces(context.Background(), ptrace.NewTraces()))
		assert.NoError(t, exporter.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
		assert.NoError(t, exporter.ConsumeLogs(context.Background(), plog.NewLogs()))
	}
}
