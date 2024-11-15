package autodetect

import (
	"context"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	"k8s.io/apimachinery/pkg/util/version"
)

type Kind string

var availableDetectors = []ClusterKindDetector{&kindDetector{}, &eksDetector{}, &gkeDetector{}, &minikubeDetector{}, &k3sDetector{}, &openshiftDetector{}, &aksDetector{}}

var currentClusterDetails ClusterDetails

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

type ClusterKindDetector interface {
	Detect(ctx context.Context, args DetectionArguments) bool
	Kind() Kind
}

type ClusterDetails struct {
	Kind       Kind
	K8SVersion *version.Version
}

func GetK8SVersion() *version.Version {
	return currentClusterDetails.K8SVersion
}

func GetClusterKind() Kind {
	return currentClusterDetails.Kind
}

func SetK8SClusterDetails(ctx context.Context, kc string, client *kube.Client)  {
	details := k8sutils.GetCurrentClusterDetails(kc)
	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		currentClusterDetails.K8SVersion = nil
	}
	ver := version.MustParse(serverVersion.String())
	currentClusterDetails.K8SVersion = ver

	args := DetectionArguments{
		ClusterDetails: details,
		ServerVersion:  serverVersion.GitVersion,
		KubeClient:     client,
	}

	for _, detector := range availableDetectors {
		relevant := detector.Detect(ctx, args)
		if !relevant {
			continue
		}
		currentClusterDetails.Kind = detector.Kind()
		return
	}

	currentClusterDetails.Kind = KindUnknown
}
