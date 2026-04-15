package otlp

import (
	"context"
)

// Consumer attaches one OTLP signal to a shared [Receiver]
type Consumer interface {
	Register(ctx context.Context, rx *Receiver) error
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

var (
	_ Consumer = (*Metrics)(nil)
	_ Consumer = (*Profiles)(nil)
)
