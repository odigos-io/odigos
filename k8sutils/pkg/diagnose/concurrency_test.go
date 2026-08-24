package diagnose

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollectionContextIndependentOfParentCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	ctx, stop := CollectionContext(parent)
	defer stop()

	cancel()

	require.NoError(t, ctx.Err(), "collection context should not be canceled when the HTTP/parent context is canceled")
	_, hasDeadline := ctx.Deadline()
	require.True(t, hasDeadline)
}

func TestCollectionContextHonorsOwnCancel(t *testing.T) {
	ctx, stop := CollectionContext(context.Background())
	stop()
	require.Error(t, ctx.Err())
}

func TestLimiterAcquireCanceledContext(t *testing.T) {
	lim := newLimiter()
	for i := 0; i < maxConcurrentK8sOps; i++ {
		require.NoError(t, lim.acquire(context.Background()))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, lim.acquire(ctx), context.Canceled)

	for i := 0; i < maxConcurrentK8sOps; i++ {
		lim.release()
	}
}

func TestLimiterAcquireRelease(t *testing.T) {
	lim := newLimiter()
	require.NoError(t, lim.acquire(context.Background()))
	lim.release()
}

func TestSleepWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	ok := sleepWithContext(ctx, time.Second)
	require.False(t, ok)
	require.Less(t, time.Since(start), 200*time.Millisecond)
}
