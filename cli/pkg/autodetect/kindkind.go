package autodetect

import (
	"context"
	"strings"
)

type kindDetector struct{}

func (k kindDetector) Detect(ctx context.Context, args DetectionArguments) (Kind, error) {
	if strings.HasPrefix(args.ClusterName, "kind-") || strings.HasPrefix(args.CurrentContext, "kind-") {
		return KindKind, nil
	}

	return KindUnknown, nil
}
