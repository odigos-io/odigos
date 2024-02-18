package verification

import (
	"context"
	"fmt"

	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/moby/moby/pkg/parsers/kernel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OdigosMinimumKernelVersion string = "4.14"
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

		for _, node := range nodeList.Items {
			nodeKernelVersion, err := kernel.ParseRelease(node.Status.NodeInfo.KernelVersion)
			if err != nil {
				return err
			}

			// nodeKernelVersion is greater than minimumKernelVersion
			if kernel.CompareKernelVersion(*nodeKernelVersion, *minimumKernelVersion) == 1 {
				return nil
			}
		}
		return fmt.Errorf(
			"verify node kernel failed, at least one node with minimum kernel version of %+q is needed",
			OdigosMinimumKernelVersion)
	}
}
