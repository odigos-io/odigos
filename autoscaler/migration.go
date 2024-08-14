package main

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Version v1.0.91
// The labels on the cluster collector deployment and the node collector daemonset
// were changed so they are both named the same and the value is the role, which might be
// extended in the future.
//
// Changes:
// - The label on the cluster collector deployment was changed from `"odigos.io/collector": "true"`
// to `"odigos.io/collector-role": "CLUSTER_GATEWAY"`.
// - The label on the node collector daemonset was changed from `"odigos.io/data-collection": "true"`
// to `"odigos.io/collector-role": "NODE_COLLECTOR"`.
//
// In k8s, `Spec.Selector.MatchLabels` for deployments and daemonsets cannot be changed after creation,
// and any update to the labels in the selector will fail with an error.
// To overcome this, we will simply delete the collectors workloads, and have them re-created with the new labels
// by the autoscaler controllers.
func MigrateCollectorsWorkloadToNewLabels(ctx context.Context, c client.Client, ns string) error {

	// Delete the cluster collector deployment itself has the label "odigos.io/collector": "true"
	// which means Spec.Selector.MatchLabels["odigos.io/collector"] = "true" as well
	preV1_0_91LabelSelectorGateway := client.MatchingLabels{"odigos.io/collector": "true"}
	err := c.DeleteAllOf(ctx, &appsv1.Deployment{}, client.InNamespace(ns), preV1_0_91LabelSelectorGateway)
	if err != nil {
		return err
	}

	// Delete the node collector daemonset itself has the label "odigos.io/data-collection": "true"
	// which means Spec.Selector.MatchLabels["odigos.io/data-collection"] = "true" as well
	preV1_0_91LabelSelectorNodeCollector := client.MatchingLabels{"odigos.io/data-collection": "true"}
	err = c.DeleteAllOf(ctx, &appsv1.DaemonSet{}, client.InNamespace(ns), preV1_0_91LabelSelectorNodeCollector)
	if err != nil {
		return err
	}

	return nil

}
