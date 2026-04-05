package collectorprofiles

import (
	"errors"

	"github.com/odigos-io/odigos/frontend/services/collector_profiles/flamegraph"
	"github.com/odigos-io/odigos/frontend/services/common"
)

// SourceIDFromStrings parses namespace, kind, and name into a SourceID.
func SourceIDFromStrings(namespace, kindStr, name string) (common.SourceID, error) {
	if namespace == "" || kindStr == "" || name == "" {
		return common.SourceID{}, errors.New("missing namespace, kind, or name")
	}
	// Same normalization as SourceKeyFromResource: UI/GraphQL may send "Deployment" vs "deployment".
	kind := NormalizeWorkloadKind(kindStr)
	return common.SourceID{Namespace: namespace, Kind: kind, Name: name}, nil
}

// EnableProfilingOutput is the result of enabling on-demand profiling for a source.
type EnableProfilingOutput struct {
	Status      string `json:"status"`
	SourceKey   string `json:"sourceKey"`
	MaxSlots    int    `json:"maxSlots"`
	ActiveSlots int    `json:"activeSlots"`
}

// EnableProfilingForSource opens or refreshes the profiling slot for a workload.
func EnableProfilingForSource(store common.ProfileStoreRef, namespace, kindStr, name string) (*EnableProfilingOutput, error) {
	id, err := SourceIDFromStrings(namespace, kindStr, name)
	if err != nil {
		return nil, err
	}
	key := SourceKeyFromSourceID(id)
	store.StartViewing(key)
	activeKeys, _ := store.DebugSlots()
	return &EnableProfilingOutput{
		Status:      "ok",
		SourceKey:   key,
		MaxSlots:    store.MaxSlots(),
		ActiveSlots: len(activeKeys),
	}, nil
}

// ReleaseProfilingOutput is returned after dropping a profiling slot.
type ReleaseProfilingOutput struct {
	Status      string `json:"status"`
	SourceKey   string `json:"sourceKey"`
	ActiveSlots int    `json:"activeSlots"`
}

// ReleaseProfilingForSource removes the in-memory slot and buffered OTLP chunks (call when the profiling UI closes).
func ReleaseProfilingForSource(store common.ProfileStoreRef, namespace, kindStr, name string) (*ReleaseProfilingOutput, error) {
	id, err := SourceIDFromStrings(namespace, kindStr, name)
	if err != nil {
		return nil, err
	}
	key := SourceKeyFromSourceID(id)
	store.RemoveSlot(key)
	activeKeys, _ := store.DebugSlots()
	return &ReleaseProfilingOutput{
		Status:      "ok",
		SourceKey:   key,
		ActiveSlots: len(activeKeys),
	}, nil
}

// GetProfilingOutput is the aggregated Pyroscope-shaped profile for a source.
type GetProfilingOutput struct {
	Profile flamegraph.FlamebearerProfile
}

// GetProfilingForSource returns the aggregated profile for a workload.
func GetProfilingForSource(store common.ProfileStoreRef, namespace, kindStr, name string) (*GetProfilingOutput, error) {
	id, err := SourceIDFromStrings(namespace, kindStr, name)
	if err != nil {
		return nil, err
	}
	key := SourceKeyFromSourceID(id)
	store.StartViewing(key)
	chunks := store.GetProfileData(key)

	if chunks == nil {
		return &GetProfilingOutput{Profile: emptyFlamebearerProfile()}, nil
	}
	return &GetProfilingOutput{Profile: BuildPyroscopeProfileFromChunks(chunks)}, nil
}

func emptyFlamebearerProfile() flamegraph.FlamebearerProfile {
	return flamegraph.FlamebearerProfile{
		Version: pyroscopeFlamebearerJSONVersion,
		Flamebearer: flamegraph.Flamebearer{
			Names:    []string{"total"},
			Levels:   [][]int64{},
			NumTicks: 0,
			MaxSelf:  0,
		},
		Metadata: pyroscopeMetadata(0),
	}
}
