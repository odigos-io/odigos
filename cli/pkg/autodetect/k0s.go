package autodetect

import (
	"context"
	"strings"
)

type k0sDetector struct{}

var _ ClusterKindDetector = &k0sDetector{}

func (k k0sDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	return strings.Contains(args.ServerVersion, "+k0s")
}

func (k k0sDetector) Kind() Kind {
	return KindK0s
}
