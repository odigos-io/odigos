package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PauseOdigos scales down instrumentor and effectively drains odiglet
// without restarting user workloads.
func PauseOdigos(ctx context.Context) error {
	ns := env.GetCurrentNamespace()

	if err := scaleDeploymentToZero(ctx, ns, k8sconsts.InstrumentorDeploymentName); err != nil {
		return fmt.Errorf("scale instrumentor to 0: %w", err)
	}

	odigletDsName := env.GetOdigletDaemonSetNameOrDefault(k8sconsts.OdigletDaemonSetName)

	if err := disableOdiglet(ctx, ns, odigletDsName); err != nil {
		return fmt.Errorf("disable odiglet: %w", err)
	}

	fmt.Printf("Paused Odigos in %q: scaled %q to 0 and patched %q\n",
		ns, k8sconsts.InstrumentorDeploymentName, odigletDsName)

	return nil
}

func scaleDeploymentToZero(ctx context.Context, ns string, name string) error {

	scale, err := kube.DefaultClient.AppsV1().Deployments(ns).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var zero int32 = 0
	scale.Spec.Replicas = zero
	_, err = kube.DefaultClient.AppsV1().Deployments(ns).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

func disableOdiglet(ctx context.Context, ns string, name string) error {
	// Patch DaemonSet template with an impossible nodeSelector and bump annotation to force rollout
	patch := []byte(`{
      "spec": {
        "updateStrategy": {
          "type": "RollingUpdate",
          "rollingUpdate": {"maxUnavailable": "100%"}
        },
        "template": {
          "metadata": {
            "annotations": {"odigos.io/pause-revision": "` + "1" + `"}
          },
          "spec": {
            "nodeSelector": {"odigos.io/disabled": "true"}
          }
        }
      }
    }`)

	if _, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return err
	}

	// Delete existing odiglet pods to stop data flow immediately (pod label is fixed "odiglet" in helm)
	selector := fmt.Sprintf("app.kubernetes.io/name=%s", k8sconsts.OdigletDaemonSetName)
	podList, err := kube.DefaultClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}

	policy := metav1.DeletePropagationBackground
	for _, pod := range podList.Items {
		_ = kube.DefaultClient.CoreV1().Pods(ns).Delete(ctx, pod.Name, metav1.DeleteOptions{PropagationPolicy: &policy})
	}

	return nil
}
