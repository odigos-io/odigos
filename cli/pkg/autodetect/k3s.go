package autodetect

import (
	"context"
	"strings"
)

type k3sDetector struct{}

func (k k3sDetector) Detect(ctx context.Context, args DetectionArguments) (Kind, error) {
	if strings.Contains(args.ServerVersion, "+k3s") {
		return KindK3s, nil
	}

	return KindUnknown, nil
}
