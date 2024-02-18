package verification

import (
	"context"
	"fmt"

	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/moby/moby/pkg/parsers/kernel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

const (
	OdigosMinimumKernelVersion string = "4.14"
)

var (
	ErrOdigosUnsupportedKernel error = fmt.Errorf(
		"verify node kernel failed, at least one node with minimum kernel version of %+q is needed",
		OdigosMinimumKernelVersion)
)

func VerifyNodeKernel(client *kube.Client) VerifierFunc {
	return func(ctx context.Context) error {
		minimumKernelVersion, err := kernel.ParseRelease(OdigosMinimumKernelVersion)
		if err != nil {
			return err
		}

		nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		return verifyNodeKernel(minimumKernelVersion, nodeList.Items)
	}
}

func verifyNodeKernel(minimumKernelVersion *kernel.VersionInfo, nodes []corev1.Node) error {
	for _, node := range nodes {
		nodeKernelVersion, err := kernel.ParseRelease(node.Status.NodeInfo.KernelVersion)
		if err != nil {
			continue
		}

		// nodeKernelVersion is greater than minimumKernelVersion
		if kernel.CompareKernelVersion(*nodeKernelVersion, *minimumKernelVersion) >= 0 {
			return nil
		}
	}

	return ErrOdigosUnsupportedKernel
}
