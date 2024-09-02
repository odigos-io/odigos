package endpoints

import (
	"context"
	"errors"
	"path"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const cdnUrl = "https://d15jtxgb40qetw.cloudfront.net"
const (
	odigosProSecretName        = "odigos-pro"
	odigosCloudTokenEnvName    = "ODIGOS_CLOUD_TOKEN"
	odigosCloudApiKeySecretKey = "odigos-cloud-api-key"
	odigosOnpremTokenEnvName   = "ODIGOS_ONPREM_TOKEN"
	odigosOnpremTokenSecretKey = "odigos-onprem-token"
)

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

func GetCurrentOdigosTier(ctx context.Context, client *kube.Client, ns string) (common.OdigosTier, error) {
	sec, err := getCurrentOdigosProSecret(ctx, client, ns)
	if err != nil {
		return "", err
	}
	if sec == nil {
		return common.CommunityOdigosTier, nil
	}

	if _, exists := sec.Data[odigosCloudApiKeySecretKey]; exists {
		return common.CloudOdigosTier, nil
	}
	if _, exists := sec.Data[odigosOnpremTokenSecretKey]; exists {
		return common.OnPremOdigosTier, nil
	}
	return common.CommunityOdigosTier, nil
}

func getCurrentOdigosProSecret(ctx context.Context, client *kube.Client, ns string) (*corev1.Secret, error) {
	secret, err := client.CoreV1().Secrets(ns).Get(ctx, odigosProSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// apparently, k8s is not setting the type meta for the object obtained with Get.
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	return secret, err
}
