package workload

// This go file contains utils to handle the kind of odigos workloads.
// it allows transforming deployments / daemonsets / statefulsets from one representation to another

// 1. the pascal case representation of the workload kind
// it is used in k8s api objects as the `Kind` field.
type WorkloadKindPascalCase string

const (
	WorkloadKindPascalCaseDeployment  WorkloadKindPascalCase = "Deployment"
	WorkloadKindPascalCaseDaemonSet   WorkloadKindPascalCase = "DaemonSet"
	WorkloadKindPascalCaseStatefulSet WorkloadKindPascalCase = "StatefulSet"
)

// 2. the lower case representation of the workload kind
// is used in odigos with the object name for instrumentation config and runtime details
type WorkloadKindLowerCase string

const (
	WorkloadKindLowerCaseDeployment  WorkloadKindLowerCase = "deployment"
	WorkloadKindLowerCaseDaemonSet   WorkloadKindLowerCase = "daemonset"
	WorkloadKindLowerCaseStatefulSet WorkloadKindLowerCase = "statefulset"
)

func WorkloadKindLowerCaseFromPascalCase(pascalCase WorkloadKindPascalCase) WorkloadKindLowerCase {
	switch pascalCase {
	case WorkloadKindPascalCaseDeployment:
		return WorkloadKindLowerCaseDeployment
	case WorkloadKindPascalCaseDaemonSet:
		return WorkloadKindLowerCaseDaemonSet
	case WorkloadKindPascalCaseStatefulSet:
		return WorkloadKindLowerCaseStatefulSet
	}
	return ""
}

func WorkloadKindPascalCaseFromLowerCase(lowerCase WorkloadKindLowerCase) WorkloadKindPascalCase {
	switch lowerCase {
	case WorkloadKindLowerCaseDeployment:
		return WorkloadKindPascalCaseDeployment
	case WorkloadKindLowerCaseDaemonSet:
		return WorkloadKindPascalCaseDaemonSet
	case WorkloadKindLowerCaseStatefulSet:
		return WorkloadKindPascalCaseStatefulSet
	}
	return ""
}
