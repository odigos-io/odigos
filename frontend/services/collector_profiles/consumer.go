package collectorprofiles

import (
	"context"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var jsonMarshaler pprofile.JSONMarshaler

// NewProfilesConsumer ingests OTLP profiles: each resource becomes one JSON chunk in the store
// when its source key is active (profiling slot open).
func NewProfilesConsumer(store *ProfileStore) (xconsumer.Profiles, error) {
	return xconsumer.NewProfiles(func(ctx context.Context, incomingBatch pprofile.Profiles) error {
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
			if !store.IsActive(sourceKey) {
				continue
			}
			appendResourceProfileJSONChunk(store, incomingBatch, resourceProfiles, idx)
		}
		return nil
	}, consumer.WithCapabilities(consumer.Capabilities{MutatesData: false}))
}

// appendResourceProfileJSONChunk extracts one resource from the batch, marshals it as OTLP Profiles JSON,
// and appends the bytes to the store slot for that resource's source key (caller ensures slot is active).
func appendResourceProfileJSONChunk(store *ProfileStore, incomingBatch pprofile.Profiles, resourceProfiles pprofile.ResourceProfilesSlice, resourceIndex int) {
	logger := commonlogger.LoggerCompat().With("subsystem", "backend-profiling")
	resourceProfile := resourceProfiles.At(resourceIndex)
	attrs := resourceProfile.Resource().Attributes()
	sourceKey, hasKey := SourceKeyFromResource(attrs)
	if !hasKey {
		return
	}
	if !store.IsActive(sourceKey) {
		return
	}
	singleResourceChunk := pprofile.NewProfiles()
	incomingBatch.Dictionary().CopyTo(singleResourceChunk.Dictionary())
	resourceProfile.CopyTo(singleResourceChunk.ResourceProfiles().AppendEmpty())
	chunkBytes, marshalErr := jsonMarshaler.MarshalProfiles(singleResourceChunk)
	if marshalErr != nil {
		logger.Warn("store_chunk", "sourceKey", sourceKey, "err", marshalErr)
		return
	}
	store.AddProfileData(sourceKey, chunkBytes)
}
