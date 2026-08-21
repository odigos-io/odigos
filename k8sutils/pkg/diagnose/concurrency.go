package diagnose

import (
	"context"
	"time"
)

type limiter struct {
	sem chan struct{}
}

func newLimiter(n int) *limiter {
	if n < 1 {
		n = 1
	}
	return &limiter{sem: make(chan struct{}, n)}
}

func (l *limiter) acquire(ctx context.Context) error {
	select {
	case l.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *limiter) release() {
	<-l.sem
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
