package instrumentation_instance

import (
	"slices"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/common/api/sampling"
)

const (
	// HeadSamplingAppliedAttribute is the non-identifying attribute in which an instrumentation
	// reports the noisy operation percentages it applied, as "<rule id>=<percent>" pairs joined by ",".
	// A process that applies no head sampling has none.
	HeadSamplingAppliedAttribute = "odigos.head_sampling.applied"

	// ConfigErrorAttribute holds the error of the last configuration the instrumentation failed to apply.
	ConfigErrorAttribute = "odigos.config.error"
)

// FormatHeadSamplingApplied renders the enabled noisy operations of a head sampling config,
// sorted by rule id so that an unchanged config renders the same.
func FormatHeadSamplingApplied(cfg *sampling.HeadSamplingConfig) string {
	if cfg == nil {
		return ""
	}
	pairs := make([]string, 0, len(cfg.NoisyOperations))
	for _, op := range cfg.NoisyOperations {
		if op.Disabled || op.Id == "" {
			continue
		}
		percent := 0.0
		if op.PercentageAtMost != nil {
			percent = *op.PercentageAtMost
		}
		pairs = append(pairs, op.Id+"="+strconv.FormatFloat(percent, 'f', -1, 64))
	}
	slices.Sort(pairs)
	return strings.Join(pairs, ",")
}

// ParseHeadSamplingApplied returns the percentage applied for each rule id.
func ParseHeadSamplingApplied(value string) map[string]float64 {
	applied := map[string]float64{}
	for _, pair := range strings.Split(value, ",") {
		id, percent, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		p, err := strconv.ParseFloat(percent, 64)
		if err != nil {
			continue
		}
		applied[id] = p
	}
	return applied
}
