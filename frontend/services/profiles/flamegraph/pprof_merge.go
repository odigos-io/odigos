package flamegraph

import (
	"sort"
	"strconv"
	"strings"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	"github.com/grafana/pyroscope/pkg/pprof"
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"
	"google.golang.org/protobuf/proto"

	"github.com/grafana/pyroscope/pkg/ingester/otlp"
)

// googleProfilesFromParsedRequest runs the same OTLP→Google conversion as ingest for every profile
// in an already-decoded ExportProfilesServiceRequest. The request dictionary must be non-nil.
func googleProfilesFromParsedRequest(req *pprofileotlp.ExportProfilesServiceRequest) []*googleProfile.Profile {
	if req == nil {
		return nil
	}
	if req.Dictionary == nil {
		req.Dictionary = &otelProfile.ProfilesDictionary{}
	}
	var out []*googleProfile.Profile
	profilesSeen := 0
	samplesSeen := 0
	samplesFirstNonZero := 0
	samplesAnyNonZero := 0
	samplesFirstZeroAnyNonZero := 0
	for _, rp := range req.ResourceProfiles {
		if rp == nil || rp.ScopeProfiles == nil {
			continue
		}
		for _, sp := range rp.ScopeProfiles {
			if sp == nil || sp.Profiles == nil {
				continue
			}
			for _, p := range sp.Profiles {
				if p == nil {
					continue
				}
				profilesSeen++
				for _, s := range p.Samples {
					if s == nil {
						continue
					}
					samplesSeen++
					firstNonZero := len(s.Values) > 0 && s.Values[0] > 0
					anyNonZero := false
					for _, v := range s.Values {
						if v > 0 {
							anyNonZero = true
							break
						}
					}
					if firstNonZero {
						samplesFirstNonZero++
					}
					if anyNonZero {
						samplesAnyNonZero++
						if !firstNonZero {
							samplesFirstZeroAnyNonZero++
						}
					}
				}
				normalizeSampleValuesForPyroscope(p)
				converted, err := otlp.ConvertOtelToGoogle(p, req.Dictionary)
				if err != nil {
					continue
				}
				for _, cp := range converted {
					cp := cp
					if gp := extractGoogleProfile(&cp); gp != nil && len(gp.Sample) > 0 {
						out = append(out, gp)
					}
				}
			}
		}
	}
	return out
}

// profileCompatibilityKey groups Google pprof profiles into merge-compatible buckets.
// Profiles in the same bucket share period type/unit, sample type list, and default sample type,
// which are the minimal schema constraints expected by pprof.ProfileMerge.
func profileCompatibilityKey(p *googleProfile.Profile) string {
	if p == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("period:")
	if p.PeriodType != nil {
		b.WriteString(stringFromPprofStringTable(p.StringTable, p.PeriodType.Type))
		b.WriteByte('/')
		b.WriteString(stringFromPprofStringTable(p.StringTable, p.PeriodType.Unit))
	}
	b.WriteString("|sampletypes:")
	for _, st := range p.SampleType {
		if st == nil {
			b.WriteString("<nil>;")
			continue
		}
		b.WriteString(stringFromPprofStringTable(p.StringTable, st.Type))
		b.WriteByte('/')
		b.WriteString(stringFromPprofStringTable(p.StringTable, st.Unit))
		b.WriteByte(';')
	}
	b.WriteString("|dst:")
	b.WriteString(strconv.FormatInt(p.DefaultSampleType, 10))
	return b.String()
}

// mergeGoogleProfilesGrouped merges compatible Google profiles per bucket using pprof.ProfileMerge.
// If a bucket fails to merge (unexpected incompatibility), samples are expanded without cross-profile merge.
func mergeGoogleProfilesGrouped(profiles []*googleProfile.Profile) (merged map[string]*googleProfile.Profile, extraSamples []Sample) {
	buckets := make(map[string][]*googleProfile.Profile)
	for _, p := range profiles {
		if p == nil {
			continue
		}
		k := profileCompatibilityKey(p)
		buckets[k] = append(buckets[k], p)
	}
	out := make(map[string]*googleProfile.Profile, len(buckets))
	for bkey, list := range buckets {
		if len(list) == 0 {
			continue
		}
		var merger pprof.ProfileMerge
		mergeOK := true
		for _, p := range list {
			pc := proto.Clone(p).(*googleProfile.Profile)
			if err := merger.Merge(pc, true); err != nil {
				mergeOK = false
				break
			}
		}
		if mergeOK {
			mp := merger.Profile()
			if mp != nil && len(mp.Sample) > 0 {
				out[bkey] = mp
			}
			continue
		}
		for _, p := range list {
			extraSamples = append(extraSamples, googleProfileToSamples(p)...)
		}
	}
	return out, extraSamples
}

func sortedKeys(m map[string]*googleProfile.Profile) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func profileTotalWeight(p *googleProfile.Profile) int64 {
	if p == nil {
		return 0
	}
	var w int64
	for _, s := range p.Sample {
		if s != nil && len(s.Value) > 0 {
			w += s.Value[0]
		}
	}
	return w
}
