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

// getNsInstrumentedLabel return the instrumentation label of the object.
// if the object is not labeled, it returns nil
func isObjectLabeledForInstrumentation(metav metav1.ObjectMeta) *bool {
	labels := metav.GetLabels()
	if labels == nil {
		return nil
	}
	namespaceInstrumented, found := labels[consts.OdigosInstrumentationLabel]
	var nsInstrumentationLabeled *bool
	if found {
		instrumentationLabel := namespaceInstrumented == consts.InstrumentationEnabled
		nsInstrumentationLabeled = &instrumentationLabel
	}
	return nsInstrumentationLabeled
}
