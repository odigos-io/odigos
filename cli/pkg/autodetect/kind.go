package autodetect

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
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

func getKindFromDetectors(ctx context.Context, args DetectionArguments) Kind {
	for _, detector := range availableDetectors {
		relevant := detector.Detect(ctx, args)
		if relevant {
			return detector.Kind()
		}
	}
	return KindUnknown
}

func getServerVersion(c *kube.Client) (string, *version.Version) {
	resp, err := c.Discovery().ServerVersion()
	if err != nil {
		fmt.Printf("Unknown k8s version, assuming oldest supported version: %s\n", k8sconsts.MinK8SVersionForInstallation)
		return k8sconsts.MinK8SVersionForInstallation.String(), k8sconsts.MinK8SVersionForInstallation
	}

	parsedVer, err := version.Parse(resp.GitVersion)
	if err != nil {
		fmt.Printf("Unknown k8s version, assuming oldest supported version: %s\n", k8sconsts.MinK8SVersionForInstallation)
		return k8sconsts.MinK8SVersionForInstallation.String(), k8sconsts.MinK8SVersionForInstallation
	}
	return resp.GitVersion, parsedVer
}

func GetK8SClusterDetails(ctx context.Context, kc string, kContext string, client *kube.Client) *ClusterDetails {
	details := k8sutils.GetCurrentClusterDetails(kc, kContext)
	serverVersionStr, serverVersion := getServerVersion(client)

	kind := getKindFromDetectors(ctx, DetectionArguments{
		ClusterDetails: details,
		ServerVersion:  serverVersionStr,
		KubeClient:     client,
	})

	return &ClusterDetails{
		Kind:       kind,
		K8SVersion: serverVersion,
	}
}
