package autodetect

import "context"

type openshiftDetector struct{}

func (o openshiftDetector) Detect(ctx context.Context, args DetectionArguments) (Kind, error) {
	apiList, err := args.KubeClient.Discovery().ServerGroups()
	if err != nil {
		return KindUnknown, err
	}

	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "route.openshift.io" {
			return KindOpenShift, nil
		}
	}

	return KindUnknown, nil
}
