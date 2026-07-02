package profiles

import (
	"context"
	"testing"
	"time"

	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestProfilesConsumerCreatesSlotOnIngest(t *testing.T) {
	store := NewProfileStore(2, 120, 1024*1024, time.Minute)
	gate := NewProfilesIngestGate(true)
	consumer, err := NewOdigosProfilesConsumer(store, gate)
	require.NoError(t, err)

	require.NoError(t, consumer.consume(context.Background(), profileBatchForSource("default", "Deployment", "checkout")))

	sourceKey := "default/Deployment/checkout"
	require.True(t, store.IsActive(sourceKey))
	require.NotEmpty(t, store.GetProfileData(sourceKey))
}

func TestProfilesConsumerDoesNotCreateSlotWhenGateDisabled(t *testing.T) {
	store := NewProfileStore(2, 120, 1024*1024, time.Minute)
	gate := NewProfilesIngestGate(false)
	consumer, err := NewOdigosProfilesConsumer(store, gate)
	require.NoError(t, err)

	require.NoError(t, consumer.consume(context.Background(), profileBatchForSource("default", "Deployment", "checkout")))

	require.False(t, store.IsActive("default/Deployment/checkout"))
}

func profileBatchForSource(namespace, kind, name string) pprofile.Profiles {
	batch := pprofile.NewProfiles()
	resourceProfiles := batch.ResourceProfiles().AppendEmpty()
	attrs := resourceProfiles.Resource().Attributes()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), namespace)
	attrs.PutStr(odigosconsts.OdigosWorkloadKindAttribute, kind)
	attrs.PutStr(odigosconsts.OdigosWorkloadNameAttribute, name)

	scopeProfiles := resourceProfiles.ScopeProfiles().AppendEmpty()
	profile := scopeProfiles.Profiles().AppendEmpty()
	profile.SampleType().SetTypeStrindex(0)
	profile.Samples().AppendEmpty().Values().Append(1)

	return batch
}
