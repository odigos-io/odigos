package autodetect

import (
	"context"
	"strings"
)

type gkeDetector struct{}

func (g gkeDetector) Detect(ctx context.Context, args DetectionArguments) (Kind, error) {
	if strings.Contains(args.ServerVersion, "-gke.") {
		return KindGKE, nil
	}

	if strings.HasPrefix(args.ClusterName, "gke_") {
		return KindGKE, nil
	}

	return KindUnknown, nil
}
