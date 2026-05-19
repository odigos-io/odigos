package odigosvmprofileattrsprocessor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

type stubProfilesConsumer struct {
	calls int
	err   error
}

func (s *stubProfilesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (s *stubProfilesConsumer) ConsumeProfiles(_ context.Context, _ pprofile.Profiles) error {
	s.calls++
	return s.err
}

func TestDropEmptyProfilesConsumer_SkipsEmptyBatch(t *testing.T) {
	next := &stubProfilesConsumer{}
	consumer := &dropEmptyProfilesConsumer{
		next:   next,
		logger: zap.NewNop(),
	}

	err := consumer.ConsumeProfiles(t.Context(), pprofile.NewProfiles())
	require.NoError(t, err)
	require.Equal(t, 0, next.calls)
}

func TestDropEmptyProfilesConsumer_ForwardsNonEmptyBatch(t *testing.T) {
	next := &stubProfilesConsumer{}
	consumer := &dropEmptyProfilesConsumer{
		next:   next,
		logger: zap.NewNop(),
	}

	profiles := pprofile.NewProfiles()
	profiles.ResourceProfiles().AppendEmpty()

	err := consumer.ConsumeProfiles(t.Context(), profiles)
	require.NoError(t, err)
	require.Equal(t, 1, next.calls)
}

func TestDropEmptyProfilesConsumer_PropagatesExporterError(t *testing.T) {
	next := &stubProfilesConsumer{err: errors.New("export failed")}
	consumer := &dropEmptyProfilesConsumer{
		next:   next,
		logger: zap.NewNop(),
	}

	profiles := pprofile.NewProfiles()
	profiles.ResourceProfiles().AppendEmpty()

	err := consumer.ConsumeProfiles(t.Context(), profiles)
	require.EqualError(t, err, "export failed")
	require.Equal(t, 1, next.calls)
}
