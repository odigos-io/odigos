package kube

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func RestartOdiglet(ctx context.Context, client *Client, ns string) error {
	// Create patch to add/update the restartedAt annotation
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`,
		time.Now().Format(time.RFC3339))

	// Patch the Odiglet daemonset
	_, err := client.AppsV1().DaemonSets(ns).Patch(
		ctx,
		k8sconsts.OdigletDaemonSetName,
		types.StrategicMergePatchType,
		[]byte(patch),
		metav1.PatchOptions{},
	)
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("odiglet daemonset not found in namespace %s", ns)
	}
	return err
}
