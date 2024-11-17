package autodetect

import (
	"context"
	"strings"
)

type k3sDetector struct{}

var _ ClusterKindDetector = &k3sDetector{}

func (k k3sDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	if strings.Contains(args.ServerVersion, "+k3s") {
		return true
	}

	return false
}

func (k k3sDetector) Kind() Kind {
	return KindK3s
}
