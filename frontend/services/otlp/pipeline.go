package otlp

import "context"

// OTLPPipeline is one OTLP signal wired into the shared UI [Receiver] (metrics or profiles currently).
type OTLPPipeline interface {
	Register(ctx context.Context) error
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
