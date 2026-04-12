package profiles

import (
	"github.com/odigos-io/odigos/frontend/services/profiles/otlpchunk"
)

// earliestProfileStartTimeUnixSec scans OTLP profile chunks (protobuf ExportProfilesServiceRequest wire)
// and returns the earliest Profile.TimeUnixNano as Unix seconds. Malformed chunks are skipped; zero
// timestamps are ignored. Returns 0 when no positive time is found.
func earliestProfileStartTimeUnixSec(chunks [][]byte) int64 {
	var minNano int64
	for _, chunk := range chunks {
		req, err := otlpchunk.UnmarshalExportProfilesRequest(chunk)
		if err != nil {
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
