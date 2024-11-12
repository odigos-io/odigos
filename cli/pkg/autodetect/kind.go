package autodetect

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
)

type Kind string

var availableDetectors = []ClusterKindDetector{&kindDetector{}, &eksDetector{}, &gkeDetector{}, &minikubeDetector{}, &k3sDetector{}, &openshiftDetector{}, &aksDetector{}}

type KubernetesVersion struct {
	Kind    Kind
	Version string
}

var CurrentKubernetesVersion KubernetesVersion

const (
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

var (
	ErrCannotDetectK8sVersion  = fmt.Errorf("cannot detect k8s version")
	ErrCannotDetectClusterKind = fmt.Errorf("cannot detect cluster kind")
)

func DetectK8SClusterDetails(ctx context.Context, kc string, client *kube.Client) (ClusterDetails, error) {
	details := k8sutils.GetCurrentClusterDetails(kc)
	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return ClusterDetails{}, errors.Join(ErrCannotDetectK8sVersion, err)
	}
	k8sVersion, err := version.NewVersion(serverVersion.GitVersion)
	if err != nil {
		return ClusterDetails{}, errors.Join(ErrCannotDetectK8sVersion, err)
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
		return ClusterDetails{
			Kind:       detector.Kind(),
			K8SVersion: k8sVersion,
		}, nil
	}

	return ClusterDetails{}, ErrCannotDetectClusterKind
}
