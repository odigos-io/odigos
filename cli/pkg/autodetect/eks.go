package autodetect

import (
	"context"
	"strings"
)

type eksDetector struct{}

var _ ClusterKindDetector = &eksDetector{}

func (e eksDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	if strings.Contains(args.ServerVersion, "-eks-") {
		return true
	}

	if strings.HasSuffix(args.ClusterName, ".eksctl.io") {
		return true
	}

	if strings.HasSuffix(args.ServerEndpoint, "eks.amazonaws.com") {
		return true
	}

	return false
}

func (e eksDetector) Kind() Kind {
	return KindEKS
}
