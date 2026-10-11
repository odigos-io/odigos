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

	if err := scaleInstrumentorDeploymentToZero(ctx, ns, k8sconsts.InstrumentorDeploymentName); err != nil {
		return fmt.Errorf("scale instrumentor to 0: %w", err)
	}
	fmt.Printf("Scaled instrumentor deployment to 0 replicas in")

	if err := disableOdiglet(ctx, ns); err != nil {
		return fmt.Errorf("disable odiglet: %w", err)
	}
	fmt.Printf("Disabled odiglet daemonset in %q\n", ns)

	return nil
}

func scaleInstrumentorDeploymentToZero(ctx context.Context, odigosNamespace string, appKubernetesName string) error {

	instrumentors, err := kube.DefaultClient.AppsV1().Deployments(odigosNamespace).List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", appKubernetesName)})
	if err != nil {
		return err
	}
	if len(instrumentors.Items) == 0 {
		return fmt.Errorf("no instrumentor deployment in namespace %q", odigosNamespace)
	}
	if len(instrumentors.Items) > 1 {
		return fmt.Errorf("multiple instrumentor deployments in namespace %q", odigosNamespace)
	}
	instrumentor := instrumentors.Items[0]
	scale, err := kube.DefaultClient.AppsV1().Deployments(odigosNamespace).GetScale(ctx, instrumentor.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var zero int32 = 0
	scale.Spec.Replicas = zero
	_, err = kube.DefaultClient.AppsV1().Deployments(odigosNamespace).UpdateScale(ctx, instrumentor.Name, scale, metav1.UpdateOptions{})
	return err
}

func disableOdiglet(ctx context.Context, ns string) error {
	selector := fmt.Sprintf("app.kubernetes.io/name=%s", k8sconsts.OdigletDaemonSetName)
	dsList, err := kube.DefaultClient.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	if len(dsList.Items) == 0 {
		return fmt.Errorf("no odiglet daemonset in namespace %q", ns)
	}
	if len(dsList.Items) > 1 {
		return fmt.Errorf("multiple odiglet daemonsets in namespace %q", ns)
	}
	ds := dsList.Items[0]

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

	if _, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Patch(ctx, ds.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return err
	}

	// Delete existing odiglet pods to stop data flow immediately (pod label is fixed "odiglet" in helm)
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
