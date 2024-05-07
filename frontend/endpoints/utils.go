package endpoints

import (
	"context"
	"errors"
	"path"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const cdnUrl = "https://d15jtxgb40qetw.cloudfront.net"

func GetImageURL(image string) string {
	return path.Join(cdnUrl, image)
}

// TODO: read this from the odigosconfig CRD
func IsSystemNamespace(namespace string) bool {
	return namespace == "kube-system" ||
		namespace == consts.DefaultNamespace ||
		namespace == "local-path-storage" ||
		namespace == "istio-system" ||
		namespace == "linkerd" ||
		namespace == "kube-node-lease"
}

func setWorkloadInstrumentationLabel(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, enabled *bool) error {
	jsonMergePatchData := getJsonMergePatchForInstrumentationLabel(enabled)

	switch workloadKind {
	case WorkloadKindDeployment:
		_, err := kube.DefaultClient.AppsV1().Deployments(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	case WorkloadKindStatefulSet:
		_, err := kube.DefaultClient.AppsV1().StatefulSets(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	case WorkloadKindDaemonSet:
		_, err := kube.DefaultClient.AppsV1().DaemonSets(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	default:
		return errors.New("unsupported workload kind " + string(workloadKind))
	}
}
