package autodetect

import "context"

type openshiftDetector struct{}

var _ ClusterKindDetector = &openshiftDetector{}

func (o openshiftDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	apiList, err := args.KubeClient.Discovery().ServerGroups()
	if err != nil {
		return false
	}

	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "route.openshift.io" {
			return true
		}
	}

	return false
}

func (o openshiftDetector) Kind() Kind {
	return KindOpenShift
}
