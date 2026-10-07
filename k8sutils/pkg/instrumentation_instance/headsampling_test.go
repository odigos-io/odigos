package instrumentation_instance

import (
	"testing"

	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/stretchr/testify/assert"
)

func TestHeadSamplingApplied(t *testing.T) {
	one, fifty := 1.0, 50.5
	cfg := &sampling.HeadSamplingConfig{NoisyOperations: []sampling.NoisyOperation{
		{Id: "b", PercentageAtMost: &fifty},
		{Id: "a", PercentageAtMost: &one},
		{Id: "c"},
		{Id: "off", Disabled: true, PercentageAtMost: &one},
		{PercentageAtMost: &one},
	}}

	value := FormatHeadSamplingApplied(cfg)
	assert.Equal(t, "a=1,b=50.5,c=0", value, "sorted by rule id; disabled rules and rules without an id are left out; no percentage drops all")
	assert.Equal(t, map[string]float64{"a": 1, "b": 50.5, "c": 0}, ParseHeadSamplingApplied(value))

	assert.Equal(t, "", FormatHeadSamplingApplied(nil))
	assert.Empty(t, ParseHeadSamplingApplied(""))
	assert.Equal(t, map[string]float64{"a": 1}, ParseHeadSamplingApplied("a=1,broken,b=x"), "malformed pairs are skipped")
}
