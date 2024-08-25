package autodetect

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
)

type Kind string

var availableDetectors = []Detector{&kindDetector{}, &eksDetector{}, &gkeDetector{}, &minikubeDetector{}, &k3sDetector{}, &openshiftDetector{}, &aksDetector{}}

type KubernetesVersion struct {
	Kind    Kind
	Version string
}

var CurrentKubernetesVersion KubernetesVersion

const (
	KindUnknown   Kind = "Unknown"
	KindMinikube  Kind = "Minikube"
	KindKind      Kind = "Kind"
	KindEKS       Kind = "EKS"
	KindGKE       Kind = "GKE"
	KindAKS       Kind = "AKS"
	KindK3s       Kind = "k3s"
	KindOpenShift Kind = "Openshift"
)

// DetectionArguments are the arguments passed to the Detect function
type DetectionArguments struct {
	k8sutils.ClusterDetails
	ServerVersion string
	KubeClient    *kube.Client
}

type Detector interface {
	Detect(ctx context.Context, args DetectionArguments) (Kind, error)
}

func KubernetesClusterProduct(ctx context.Context, kc string, client *kube.Client) (Kind, string) {
	details := k8sutils.GetCurrentClusterDetails(kc)
	serverVersion, err := client.Discovery().ServerVersion()
	kubeVersion := fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor)
	gitServerVersion := ""
	if err == nil {
		gitServerVersion = serverVersion.GitVersion
	}

	args := DetectionArguments{
		ClusterDetails: details,
		ServerVersion:  gitServerVersion,
		KubeClient:     client,
	}

	for _, detector := range availableDetectors {
		kind, err := detector.Detect(ctx, args)
		if err != nil {
			continue
		}
		if kind != KindUnknown {
			return kind, kubeVersion
		}
	}

	return KindUnknown, kubeVersion
}
