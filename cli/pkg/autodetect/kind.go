package autodetect

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"k8s.io/apimachinery/pkg/util/version"
)

type Kind string

var availableDetectors = []ClusterKindDetector{&kindDetector{}, &eksDetector{}, &gkeDetector{}, &minikubeDetector{}, &k3sDetector{}, &openshiftDetector{}, &aksDetector{}}

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

func GetK8SClusterDetails(ctx context.Context, kc string, kContext string, client *kube.Client) *ClusterDetails {
	clusterDetails := &ClusterDetails{}
	details := k8sutils.GetCurrentClusterDetails(kc, kContext)
	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		clusterDetails.K8SVersion = nil
		fmt.Printf("Unknown k8s version, assuming oldest supported version: %s\n", k8sconsts.MinK8SVersionForInstallation)
	} else {
		ver := version.MustParse(serverVersion.String())
		clusterDetails.K8SVersion = ver
	}

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
		clusterDetails.Kind = detector.Kind()
		return clusterDetails
	}

	clusterDetails.Kind = KindUnknown
	return clusterDetails
}
