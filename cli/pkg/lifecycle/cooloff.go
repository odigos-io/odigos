package lifecycle

import (
	"context"
	"time"
)

func SetCoolOff(ctx context.Context, coolOff time.Duration) context.Context {
	return context.WithValue(ctx, "coolOff", coolOff)
}

func GetCoolOff(ctx context.Context) time.Duration {
	coolOff := ctx.Value("coolOff")
	if coolOff == nil {
		return 0
	}

	coolOffDuration, ok := coolOff.(time.Duration)
	if !ok {
		return 0
	}
	return coolOffDuration
}
