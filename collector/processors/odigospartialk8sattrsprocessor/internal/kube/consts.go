package kube

// WorkloadKind represents the kind of Kubernetes workload
type WorkloadKind string

const (
	WorkloadKindDeployment       WorkloadKind = "Deployment"
	WorkloadKindDaemonSet        WorkloadKind = "DaemonSet"
	WorkloadKindStatefulSet      WorkloadKind = "StatefulSet"
	WorkloadKindCronJob          WorkloadKind = "CronJob"
	WorkloadKindJob              WorkloadKind = "Job"
	WorkloadKindDeploymentConfig WorkloadKind = "DeploymentConfig"
	WorkloadKindArgoRollout      WorkloadKind = "Rollout"
)

// K8SArgoRolloutNameAttribute is the attribute key for Argo Rollout name
const K8SArgoRolloutNameAttribute = "k8s.argoproj.rollout.name"

// argoRolloutUniqueLabelKey is the default key of the selector that is added
// to rollout pods. This is used to detect if a pod belongs to an Argo Rollout.
// Value from: github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1
const argoRolloutUniqueLabelKey = "rollouts-pod-template-hash"
