package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOdigosConfiguration_ProfilingEnabled(t *testing.T) {
	assert.False(t, (*OdigosConfiguration)(nil).ProfilingEnabled())

	cfg := &OdigosConfiguration{}
	assert.False(t, cfg.ProfilingEnabled())

	off := false
	cfg.Profiling = &ProfilingConfiguration{Enabled: &off}
	assert.False(t, cfg.ProfilingEnabled())

	on := true
	cfg.Profiling.Enabled = &on
	assert.True(t, cfg.ProfilingEnabled())
}

func TestProfilingPipelineActive(t *testing.T) {
	assert.False(t, ProfilingPipelineActive(nil))

	p := &ProfilingConfiguration{}
	assert.False(t, ProfilingPipelineActive(p))

	disabled := false
	p.Enabled = &disabled
	assert.False(t, ProfilingPipelineActive(p))

	enabled := true
	p.Enabled = &enabled
	assert.True(t, ProfilingPipelineActive(p))
}
