package autodetect

import (
	"context"
	"strings"
)

type gkeDetector struct{}

var _ ClusterKindDetector = &gkeDetector{}

func (g gkeDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	if strings.Contains(args.ServerVersion, "-gke.") {
		return true
	}

	if strings.HasPrefix(args.ClusterName, "gke_") {
		return true
	}

	return false
}

func (g gkeDetector) Kind() Kind {
	return KindGKE
}
