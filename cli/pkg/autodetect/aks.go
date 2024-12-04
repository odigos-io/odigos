package autodetect

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type aksDetector struct{}

var _ ClusterKindDetector = &aksDetector{}

func (a aksDetector) Detect(ctx context.Context, args DetectionArguments) bool {
	if strings.HasSuffix(args.ServerEndpoint, "azmk8s.io:443") {
		return true
	}

	// Look for nodes that have an AKS specific label
	listOpts := metav1.ListOptions{
		LabelSelector: "kubernetes.azure.com/cluster",
		// Only need one
		Limit: 1,
	}

	nodes, err := args.KubeClient.CoreV1().Nodes().List(ctx, listOpts)
	if err != nil {
		return false
	}
	if len(nodes.Items) > 0 {
		return true
	}

	return false
}

func (a aksDetector) Kind() Kind {
	return KindAKS
}
