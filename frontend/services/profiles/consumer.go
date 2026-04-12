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
	store *ProfileStore
	otlp  xconsumer.Profiles
}

// NewOdigosProfilesConsumer builds a profiles consumer for the given store.
func NewOdigosProfilesConsumer(store *ProfileStore) (*OdigosProfilesConsumer, error) {
	c := &OdigosProfilesConsumer{store: store}
	otlp, err := xconsumer.NewProfiles(
		c.consume,
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
	if err != nil {
		return nil, err
	}
	c.otlp = otlp
	return c, nil
}

func (c *OdigosProfilesConsumer) OTLPProfiles() xconsumer.Profiles {
	return c.otlp
}

func (c *OdigosProfilesConsumer) consume(ctx context.Context, incomingBatch pprofile.Profiles) error {
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
		appendResourceProfileChunk(c.store, incomingBatch, resourceProfiles, idx)
	}
	return nil
}

// appendResourceProfileChunk extracts one resource from the batch, marshals it as OTLP Profiles
// protobuf (ExportProfilesServiceRequest wire, pdata ProtoMarshaler), and appends to the store slot.
func appendResourceProfileChunk(store *ProfileStore, incomingBatch pprofile.Profiles, resourceProfiles pprofile.ResourceProfilesSlice, resourceIndex int) {
	log := commonlogger.LoggerCompat().With("subsystem", "backend-profiling")
	resourceProfile := resourceProfiles.At(resourceIndex)
	attrs := resourceProfile.Resource().Attributes()
	sourceKey, hasKey := SourceKeyFromResource(attrs)
	if !hasKey {
		return
	}
	if !store.IsActive(sourceKey) {
		return
	}
	singleResourceChunk := buildSingleResourceProfilesFromBatch(incomingBatch, resourceProfiles, resourceIndex)
	chunkBytes, marshalErr := protoMarshaler.MarshalProfiles(singleResourceChunk)
	if marshalErr != nil {
		log.Warn("store_chunk", "sourceKey", sourceKey, "err", marshalErr)
		return
	}
	store.AddProfileData(sourceKey, chunkBytes)
}

// buildSingleResourceProfilesFromBatch builds a standalone pprofile.Profiles message
// holding one ResourceProfiles entry from the batch.
func buildSingleResourceProfilesFromBatch(
	incomingBatch pprofile.Profiles,
	resourceProfiles pprofile.ResourceProfilesSlice,
	resourceIndex int,
) pprofile.Profiles {
	out := pprofile.NewProfiles()
	// Copy the full batch dictionary: OTLP Profiles store strings, stacks, mappings, etc.
	// In ProfilesDictionary and nested data references those tables by index.
	incomingBatch.Dictionary().CopyTo(out.Dictionary())
	resourceProfiles.At(resourceIndex).CopyTo(out.ResourceProfiles().AppendEmpty())
	return out
}
