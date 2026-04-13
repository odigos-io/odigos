package clustercollector

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddProfilingGatewayPipeline_Disabled(t *testing.T) {
	var c config.Config
	err := addProfilingGatewayPipeline(&c, "odigos-system", nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, c.Processors)

	off := false
	err = addProfilingGatewayPipeline(&c, "odigos-system", &common.ProfilingConfiguration{Enabled: &off}, nil)
	assert.NoError(t, err)
	assert.Nil(t, c.Processors)
}

func TestAddProfilingGatewayPipeline_Enabled(t *testing.T) {
	on := true
	var c config.Config
	gw := &odigosv1.CollectorsGroup{
		Spec: odigosv1.CollectorsGroupSpec{
			ResourcesSettings: odigosv1.CollectorsGroupResourcesSettings{
				MemoryLimiterLimitMiB:      400,
				MemoryLimiterSpikeLimitMiB: 80,
			},
		},
	}
	err := addProfilingGatewayPipeline(&c, "odigos-system", &common.ProfilingConfiguration{Enabled: &on}, gw)
	require.NoError(t, err)

	require.NotNil(t, c.Processors)
	assert.Contains(t, c.Processors, "memory_limiter")
	assert.Contains(t, c.Processors, odigosconsts.GenericBatchProcessorConfigKey)

	pl := c.Service.Pipelines["profiles"]
	assert.Equal(t, []string{"otlp"}, pl.Receivers)
	assert.Equal(t, []string{"memory_limiter", odigosconsts.GenericBatchProcessorConfigKey}, pl.Processors)
	assert.Equal(t, []string{commonconf.ProfilingGatewayToUIExporter}, pl.Exporters)
}
