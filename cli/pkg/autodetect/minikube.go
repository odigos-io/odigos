package autodetect

import "context"

type minikubeDetector struct{}

func (m minikubeDetector) Detect(ctx context.Context, args DetectionArguments) (Kind, error) {
	if args.ClusterName == "minikube" {
		return KindMinikube, nil
	}

	if args.CurrentContext == "minikube" {
		return KindMinikube, nil
	}

	return KindUnknown, nil
}
