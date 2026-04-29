package profiles

import (
	"context"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var protoMarshaler pprofile.ProtoMarshaler

type OdigosProfilesConsumer struct {
	store    *ProfileStore
	gate     *IngestGate
	profiles xconsumer.Profiles
}

// NewOdigosProfilesConsumer builds a profiles consumer for the given store.
// gate controls whether incoming OTLP profiles are stored; when false, batches are dropped.
func NewOdigosProfilesConsumer(store *ProfileStore, gate *IngestGate) (*OdigosProfilesConsumer, error) {
	profilesConsumer := &OdigosProfilesConsumer{store: store, gate: gate}
	profiles, err := xconsumer.NewProfiles(
		profilesConsumer.consume,
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
	if err != nil {
		return nil, err
	}
	profilesConsumer.profiles = profiles
	return profilesConsumer, nil
}

func (c *OdigosProfilesConsumer) GetConsumer() xconsumer.Profiles {
	return c.profiles
}

func (c *OdigosProfilesConsumer) consume(ctx context.Context, incomingBatch pprofile.Profiles) error {
	// IngestGate is flipped by k8s watcher watching the effective-config
	// The OTLP profiles receiver stays up, when the gate is off we drop batches here so the UI pod does not need a restart.
	if c.gate != nil && !c.gate.IsEnabled() {
		return nil
	}
	resourceProfiles := incomingBatch.ResourceProfiles()
	numResources := resourceProfiles.Len()
	if numResources == 0 {
		return nil
	}

	for idx := 0; idx < numResources; idx++ {
		attrs := resourceProfiles.At(idx).Resource().Attributes()
		sourceKey, ok := SourceKeyFromResource(attrs)
		if !ok {
			continue
		}
		if !c.store.IsActive(sourceKey) {
			continue
		}
		appendResourceProfileChunk(c.store, sourceKey, incomingBatch, resourceProfiles, idx)
	}
	return nil
}

// appendResourceProfileChunk marshals a single resource's profiles as OTLP protobuf and appends to the slot
func appendResourceProfileChunk(store *ProfileStore, sourceKey string, incomingBatch pprofile.Profiles, resourceProfiles pprofile.ResourceProfilesSlice, resourceIndex int) {
	log := commonlogger.LoggerCompat().With("subsystem", "backend-profiling")
	singleResourceChunk := buildSingleResourceProfilesFromBatch(incomingBatch, resourceProfiles, resourceIndex)
	chunkBytes, marshalErr := protoMarshaler.MarshalProfiles(singleResourceChunk)
	if marshalErr != nil {
		log.Warn("store_chunk", "sourceKey", sourceKey, "err", marshalErr)
		return
	}
	store.AddProfileData(sourceKey, chunkBytes)
}

// buildSingleResourceProfilesFromBatch builds a standalone pprofile.Profiles message holding one ResourceProfiles entry from the batch.
func buildSingleResourceProfilesFromBatch(
	incomingBatch pprofile.Profiles,
	resourceProfiles pprofile.ResourceProfilesSlice,
	resourceIndex int,
) pprofile.Profiles {
	out := pprofile.NewProfiles()
	// Copy the full batch dictionary into each stored chunk
	// OTLP Profiles wire format uses a shared
	// dictionary (string table, mappings, attribute tables).
	incomingBatch.Dictionary().CopyTo(out.Dictionary())
	resourceProfiles.At(resourceIndex).CopyTo(out.ResourceProfiles().AppendEmpty())
	return out
}
