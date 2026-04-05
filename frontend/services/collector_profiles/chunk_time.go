package collectorprofiles

import (
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/protobuf/encoding/protojson"
)

// earliestProfileStartTimeUnixSec scans OTLP JSON chunks for the smallest profile TimeUnixNano
// to populate Flamebearer timeline start (seconds). Required for correct time axis when merging chunks.
// Implemented here (not utils.go) because it unmarshals OTLP profile protobuf JSON; utils.go only maps attributes.
func earliestProfileStartTimeUnixSec(chunks [][]byte) int64 {
	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}
	var minNano int64
	for _, chunk := range chunks {
		req := &pprofileotlp.ExportProfilesServiceRequest{}
		if unmarshal.Unmarshal(chunk, req) != nil {
			continue
		}
		for _, rp := range req.ResourceProfiles {
			for _, sp := range rp.ScopeProfiles {
				for _, prof := range sp.Profiles {
					nano := int64(prof.TimeUnixNano)
					if nano > 0 && (minNano == 0 || nano < minNano) {
						minNano = nano
					}
				}
			}
		}
	}
	if minNano == 0 {
		return 0
	}
	return minNano / 1e9
}
