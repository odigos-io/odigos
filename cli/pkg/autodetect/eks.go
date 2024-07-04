package autodetect

import (
	"context"
	"strings"
)

type eksDetector struct{}

func (e eksDetector) Detect(ctx context.Context, args DetectionArguments) (Kind, error) {
	if strings.Contains(args.ServerVersion, "-eks-") {
		return KindEKS, nil
	}

	if strings.HasSuffix(args.ClusterName, ".eksctl.io") {
		return KindEKS, nil
	}

	if strings.HasSuffix(args.ServerEndpoint, "eks.amazonaws.com") {
		return KindEKS, nil
	}

	return KindUnknown, nil
}
