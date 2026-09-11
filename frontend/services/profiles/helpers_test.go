package profiles

import (
	"testing"
	"time"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/profilecache"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

const (
	profilesTestNamespace = "checkout-ns"
	profilesTestWorkload  = "checkout-service"
)

func newProfilesTestStore(t *testing.T) *ProfileStore {
	t.Helper()
	return profilecache.NewStore(profilecache.StoreConfig{
		MaxSlots:        4,
		TTLSeconds:      600,
		SlotMaxBytes:    1 << 20,
		CleanupInterval: time.Hour,
	})
}

// workloadProfile describes one resource's profile inside a test OTLP batch.
type workloadProfile struct {
	namespace string
	kind      string
	name      string
	timeNano  uint64
	// frames is a single sample's stack, root-first.
	frames []string
	value  int64
}

// otlpProfilesBatch builds an OTLP profiles batch the way an agent would: symbols live in the
// shared dictionary and each sample references a stack by index.
func otlpProfilesBatch(t *testing.T, workloads ...workloadProfile) pprofile.Profiles {
	t.Helper()

	batch := pprofile.NewProfiles()
	dictionary := batch.Dictionary()
	dictionary.StringTable().Append("")
	// Locations default to mapping index 0, so the table needs an entry for symbol resolution.
	dictionary.MappingTable().AppendEmpty()

	stringIdx := map[string]int32{"": 0}
	intern := func(s string) int32 {
		if idx, ok := stringIdx[s]; ok {
			return idx
		}
		dictionary.StringTable().Append(s)
		idx := int32(dictionary.StringTable().Len() - 1)
		stringIdx[s] = idx
		return idx
	}

	locationIdx := map[string]int32{}
	locationForFrame := func(frame string) int32 {
		if idx, ok := locationIdx[frame]; ok {
			return idx
		}
		fn := dictionary.FunctionTable().AppendEmpty()
		fn.SetNameStrindex(intern(frame))
		fnIdx := int32(dictionary.FunctionTable().Len() - 1)

		loc := dictionary.LocationTable().AppendEmpty()
		loc.Lines().AppendEmpty().SetFunctionIndex(fnIdx)
		idx := int32(dictionary.LocationTable().Len() - 1)
		locationIdx[frame] = idx
		return idx
	}

	// OTLP stores location indices leaf-first.
	addStack := func(rootFirstFrames []string) int32 {
		stack := dictionary.StackTable().AppendEmpty()
		for i := len(rootFirstFrames) - 1; i >= 0; i-- {
			stack.LocationIndices().Append(locationForFrame(rootFirstFrames[i]))
		}
		return int32(dictionary.StackTable().Len() - 1)
	}

	for _, w := range workloads {
		rp := batch.ResourceProfiles().AppendEmpty()
		attrs := rp.Resource().Attributes()
		attrs.PutStr(string(semconv.K8SNamespaceNameKey), w.namespace)
		attrs.PutStr(consts.OdigosWorkloadKindAttribute, w.kind)
		attrs.PutStr(consts.OdigosWorkloadNameAttribute, w.name)

		prof := rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
		// A non-CPU profile type keeps sample values from being rescaled by the period.
		prof.SampleType().SetTypeStrindex(intern("alloc_space"))
		prof.SampleType().SetUnitStrindex(intern("bytes"))
		prof.PeriodType().SetTypeStrindex(intern("space"))
		prof.PeriodType().SetUnitStrindex(intern("bytes"))
		prof.SetPeriod(1)
		prof.SetTime(pcommon.Timestamp(w.timeNano))

		sample := prof.Samples().AppendEmpty()
		sample.SetStackIndex(addStack(w.frames))
		sample.Values().Append(w.value)
	}

	return batch
}

func checkoutWorkloadProfile(frames []string, value int64) workloadProfile {
	return workloadProfile{
		namespace: profilesTestNamespace,
		kind:      "Deployment",
		name:      profilesTestWorkload,
		timeNano:  1_700_000_000_000_000_000,
		frames:    frames,
		value:     value,
	}
}

// otlpChunk marshals a batch the same way the consumer stores it, so tests can feed the stored
// wire format back into the flamegraph parser.
func otlpChunk(t *testing.T, batch pprofile.Profiles) []byte {
	t.Helper()
	raw, err := pprofileotlp.NewExportRequestFromProfiles(batch).MarshalProto()
	require.NoError(t, err)
	return raw
}
