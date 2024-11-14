package autodetect

import (
	"context"
	"strings"
)

type kindDetector struct{}

var _ ClusterKindDetector = &kindDetector{}

func (k kindDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	if strings.HasPrefix(args.ClusterName, "kind-") || strings.HasPrefix(args.CurrentContext, "kind-") {
		return true
	}

	return false
}

func (k kindDetector) Kind() Kind {
	return KindKind
}
