package flamegraph

import (
	pyrofb "github.com/grafana/pyroscope/pkg/og/structs/flamebearer"
)

// AdaptPyroscopeFlamebearerProfile maps Grafana Pyroscope's flamebearer export into the JSON shape we return over GraphQL.
// Reuses upstream profile structs and only adds Odigos-only symbols.
func AdaptPyroscopeFlamebearerProfile(
	up *pyrofb.FlamebearerProfile,
	timeline *pyrofb.FlamebearerTimelineV1,
	symbols []SymbolStats,
) FlamebearerProfile {
	if up == nil {
		up = &pyrofb.FlamebearerProfile{
			Version: 1,
			FlamebearerProfileV1: pyrofb.FlamebearerProfileV1{
				Flamebearer: pyrofb.FlamebearerV1{
					Names:    []string{"total"},
					Levels:   [][]int{},
					NumTicks: 0,
					MaxSelf:  0,
				},
			},
		}
	}
	up.Timeline = timeline
	return FlamebearerProfile{
		FlamebearerProfile: up,
		Symbols:            symbols,
	}
}
