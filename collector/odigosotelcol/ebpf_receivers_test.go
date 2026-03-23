package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	ebpfcollector "go.opentelemetry.io/ebpf-profiler/collector"

	odigosebpfreceiver "github.com/odigos-io/odigos/collector/receivers/odigosebpfreceiver"
)

func TestEbpfReceiversRegisteredInDistribution(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	odigo := odigosebpfreceiver.NewFactory().Type()
	prof := ebpfcollector.NewFactory().Type()
	require.Equal(t, component.MustNewType("odigosebpf"), odigo)
	require.Equal(t, component.MustNewType("profiling"), prof)
	require.NotEqual(t, odigo, prof)
	require.Contains(t, factories.Receivers, odigo)
	require.Contains(t, factories.Receivers, prof)
}
