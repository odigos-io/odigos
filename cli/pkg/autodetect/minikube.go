package autodetect

import "context"

type minikubeDetector struct{}

var _ ClusterKindDetector = &minikubeDetector{}

func (m minikubeDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	if args.ClusterName == "minikube" {
		return true
	}

	if args.CurrentContext == "minikube" {
		return true
	}

	return false
}

func (m minikubeDetector) Kind() Kind {
	return KindMinikube
}
