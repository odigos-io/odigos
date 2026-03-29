package clustercollector

import (
	"testing"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddProfilingGatewayPipeline_Disabled(t *testing.T) {
	var c config.Config
	err := addProfilingGatewayPipeline(&c, "odigos-system", nil)
	assert.NoError(t, err)
	assert.Nil(t, c.Processors)

	off := false
	err = addProfilingGatewayPipeline(&c, "odigos-system", &common.ProfilingConfiguration{Enabled: &off})
	assert.NoError(t, err)
	assert.Nil(t, c.Processors)
}

func TestAddProfilingGatewayPipeline_Enabled(t *testing.T) {
	on := true
	var c config.Config
	err := addProfilingGatewayPipeline(&c, "odigos-system", &common.ProfilingConfiguration{Enabled: &on})
	require.NoError(t, err)

	require.Contains(t, c.Processors, filterProfilesGatewayProcessor)
	filterCfg, ok := c.Processors[filterProfilesGatewayProcessor].(config.GenericMap)
	require.True(t, ok)
	assert.Equal(t, commonconf.ProfilingFilterProcessorConfig(), filterCfg)

	pl := c.Service.Pipelines["profiles"]
	assert.Equal(t, []string{"otlp"}, pl.Receivers)
	assert.Equal(t, []string{filterProfilesGatewayProcessor, k8sAttributesProfilesGatewayProcessor}, pl.Processors)
	assert.Equal(t, []string{otlpProfilesToUIExporterName}, pl.Exporters)
}
