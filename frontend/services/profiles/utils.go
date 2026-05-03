package profiles

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// SourceKeyFromResource extracts namespace, kind and name from OTLP resource attributes.
func SourceKeyFromResource(attrs pcommon.Map) (string, bool) {
	sID, err := common.ResourceAttributesToSourceID(attrs)
	if err != nil || sID.Name == "" {
		return "", false
	}
	return sID.Namespace + "/" + string(sID.Kind) + "/" + sID.Name, true
}

func NormalizeWorkloadKind(kindStr string) k8sconsts.WorkloadKind {
	if k := workload.WorkloadKindFromString(kindStr); k != "" {
		return k
	}
	return k8sconsts.WorkloadKind(kindStr)
}

// SourceKeyFromSourceID returns a stable string key for the given SourceID.
func SourceKeyFromSourceID(id common.SourceID) string {
	return id.Namespace + "/" + string(id.Kind) + "/" + id.Name
}

// earliestProfileStartTimeUnixSec scans OTLP profile chunks for the smallest
// profile TimeUnixNano to populate Flamebearer timeline start (seconds).
// Required for correct time axis when merging chunks.
func earliestProfileStartTimeUnixSec(chunks [][]byte) int64 {
	var minNano int64
	for _, chunk := range chunks {
		req, err := flamegraph.ParseExportProfilesServiceRequest(chunk)
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
	return minNano / 1e9
}
