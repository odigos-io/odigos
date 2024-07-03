package autodetect

import (
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
)

type Kind string

var availableDetectors = []Detector{&gkeDetector{}}

const (
	KindUnknown   Kind = "Unknown"
	KindMinikube  Kind = "Minikube"
	KindKind      Kind = "KinD"
	KindEKS       Kind = "EKS"
	KindGKE       Kind = "GKE"
	KindAKS       Kind = "AKS"
	KindDocker    Kind = "Docker"
	KindK3s       Kind = "k3s"
	KindOpenShift Kind = "Openshift"
)

// DetectionArguments are the arguments passed to the Detect function
type DetectionArguments struct {
	// CurrentContext is the current kubectl context
	CurrentContext string
	// ServerVersion is the Kubernetes server version - this is used by multiple detectors
	ServerVersion string
	// KubeClient is a Kubernetes client
	KubeClient *kube.Client
}

type Detector interface {
	Detect(args DetectionArguments) (Kind, error)
}

func KubernetesClusterProduct(kc string, client *kube.Client) (Kind, string) {
	currentCtx := k8sutils.GetCurrentContext(kc)
	serverVersion, err := client.Discovery().ServerVersion()
	kubeVersion := fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor)
	gitServerVersion := ""
	if err == nil {
		gitServerVersion = serverVersion.GitVersion
	}

	args := DetectionArguments{
		CurrentContext: currentCtx,
		ServerVersion:  gitServerVersion,
		KubeClient:     client,
	}

	for _, detector := range availableDetectors {
		kind, err := detector.Detect(args)
		if err != nil {
			continue
		}
		if kind != KindUnknown {
			return kind, kubeVersion
		}
	}

	return KindUnknown, kubeVersion
}
