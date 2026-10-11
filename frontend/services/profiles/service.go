package profiles

import (
	"context"
	"errors"

	pyrofb "github.com/grafana/pyroscope/pkg/og/structs/flamebearer"
	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
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
	store.EnsureSlot(key)
	activeKeys, _ := store.ActiveSlots()
	return &EnableProfilingOutput{
		Status:      "ok",
		SourceKey:   key,
		MaxSlots:    store.MaxSlots(),
		ActiveSlots: len(activeKeys),
	}, nil
}

// DisableProfilingOutput is returned after a profiling slot is disabled and its memory freed.
type DisableProfilingOutput struct {
	Status      string `json:"status"`
	SourceKey   string `json:"sourceKey"`
	ActiveSlots int    `json:"activeSlots"`
}

// ClearProfilingBufferOutput is returned after clearing buffered OTLP chunks for a slot.
type ClearProfilingBufferOutput struct {
	Status      string `json:"status"`
	SourceKey   string `json:"sourceKey"`
	ActiveSlots int    `json:"activeSlots"`
}

// ClearProfilingBufferForSource empties the in-memory profile chunks for a workload but keeps
// the profiling slot active so new samples can accumulate.
func ClearProfilingBufferForSource(store common.ProfileStoreRef, namespace, kindStr, name string) (*ClearProfilingBufferOutput, error) {
	id, err := SourceIDFromStrings(namespace, kindStr, name)
	if err != nil {
		return nil, err
	}
	key := SourceKeyFromSourceID(id)
	if !store.ClearSlotBuffer(key) {
		return nil, errors.New("no active profiling slot for this source")
	}
	activeKeys, _ := store.ActiveSlots()
	return &ClearProfilingBufferOutput{
		Status:      "ok",
		SourceKey:   key,
		ActiveSlots: len(activeKeys),
	}, nil
}

// DisableProfilingForSource removes the in-memory slot and buffered OTLP chunks (call when the profiling UI closes).
func DisableProfilingForSource(store common.ProfileStoreRef, namespace, kindStr, name string) (*DisableProfilingOutput, error) {
	id, err := SourceIDFromStrings(namespace, kindStr, name)
	if err != nil {
		return nil, err
	}
	key := SourceKeyFromSourceID(id)
	store.RemoveSlot(key)
	activeKeys, _ := store.ActiveSlots()
	return &DisableProfilingOutput{
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
func GetProfilingForSource(ctx context.Context, store common.ProfileStoreRef, namespace, kindStr, name string) (*GetProfilingOutput, error) {
	id, err := SourceIDFromStrings(namespace, kindStr, name)
	if err != nil {
		return nil, err
	}
	key := SourceKeyFromSourceID(id)
	store.EnsureSlot(key)
	chunks := store.GetProfileData(key)
	if chunks == nil {
		return &GetProfilingOutput{Profile: emptyFlamebearerProfile()}, nil
	}
	return &GetProfilingOutput{Profile: buildPyroscopeProfileFromChunks(ctx, chunks)}, nil
}

func emptyFlamebearerProfile() flamegraph.FlamebearerProfile {
	return flamegraph.FlamebearerProfile{
		FlamebearerProfile: &pyrofb.FlamebearerProfile{
			Version: pyroscopeFlamebearerJSONVersion,
			FlamebearerProfileV1: pyrofb.FlamebearerProfileV1{
				Flamebearer: pyrofb.FlamebearerV1{
					Names:    []string{"total"},
					Levels:   [][]int{},
					NumTicks: 0,
					MaxSelf:  0,
				},
				Metadata: pyroscopeMetadata(),
			},
		},
	}
}
