package collectors

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetOdigletDaemonSetInfo(ctx context.Context) (*model.CollectorDaemonSetInfo, error) {
	ns := env.GetCurrentNamespace()
	name := env.GetOdigletDaemonSetNameOrDefault(k8sconsts.OdigletDaemonSetName)
	ds, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	status, inProgress := computeDaemonSetStatus(ds)
	nodes := &model.NodesSummary{Desired: int(ds.Status.DesiredNumberScheduled), Ready: int(ds.Status.NumberReady)}

	result := &model.CollectorDaemonSetInfo{Status: status, Nodes: nodes, RolloutInProgress: inProgress, ImageVersion: services.StringPtr(extractImageVersionForContainer(ds.Spec.Template.Spec.Containers, k8sconsts.OdigosNodeCollectorContainerName)), LastRolloutAt: services.StringPtr(findDaemonSetLastRolloutTime(ctx, ds))}

	if rr := extractResourcesForContainer(ds.Spec.Template.Spec.Containers, k8sconsts.OdigosNodeCollectorContainerName); rr != nil {
		result.Resources = rr
	}

	manifestYAML, err := services.K8sManifest(ctx, ns, model.K8sResourceKindDaemonSet, name)
	if err != nil {
		return nil, err
	}
	result.ManifestYaml = manifestYAML

	configMapYAML, err := services.K8sManifest(ctx, ns, model.K8sResourceKindConfigMap, k8sconsts.OdigosNodeCollectorConfigMapName)
	if err != nil {
		return nil, err
	}
	result.ConfigMapYaml = configMapYAML

	return result, nil
}

func computeDaemonSetStatus(ds *appsv1.DaemonSet) (model.WorkloadRolloutStatus, bool) {
	desired := ds.Status.DesiredNumberScheduled
	updated := ds.Status.UpdatedNumberScheduled
	available := ds.Status.NumberAvailable
	ready := ds.Status.NumberReady
	unavailable := ds.Status.NumberUnavailable

	if available == 0 {
		return model.WorkloadRolloutStatusDown, updated < desired || available < desired
	}
	if updated < desired || available < desired {
		if unavailable > 0 {
			return model.WorkloadRolloutStatusDegraded, true
		}
		return model.WorkloadRolloutStatusUpdating, true
	}
	if desired == updated && desired == available && desired == ready && unavailable == 0 {
		return model.WorkloadRolloutStatusHealthy, false
	}

	return model.WorkloadRolloutStatusUnknown, false
}

func findDaemonSetLastRolloutTime(ctx context.Context, ds *appsv1.DaemonSet) string {
	if ds.Spec.Template.Annotations != nil {
		if v, ok := ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]; ok {
			return v
		}
	}
	revList, err := kube.DefaultClient.AppsV1().ControllerRevisions(ds.Namespace).List(ctx, metav1.ListOptions{LabelSelector: k8sconsts.OdigosCollectorRoleLabel + "=" + string(k8sconsts.CollectorsRoleNodeCollector)})
	if err == nil && len(revList.Items) > 0 {
		var latest metav1.Time
		for _, cr := range revList.Items {
			for _, o := range cr.OwnerReferences {
				if o.Kind == "DaemonSet" && o.UID == ds.UID {
					if latest.IsZero() || cr.CreationTimestamp.After(latest.Time) {
						latest = cr.CreationTimestamp
					}
					break
				}
			}
		}
		if !latest.IsZero() {
			return services.Metav1TimeToString(latest)
		}
	}
	return services.Metav1TimeToString(ds.CreationTimestamp)
}
