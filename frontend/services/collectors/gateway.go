package collectors

import (
	"context"
	"fmt"
	"log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetGatewayDeploymentInfo fetches Deployment/HPA and computes the header info for the gateway deployment.
func GetGatewayDeploymentInfo(ctx context.Context) (*model.GatewayDeploymentInfo, error) {
	ns := env.GetCurrentNamespace()

	depList, err := kube.DefaultClient.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{LabelSelector: k8sconsts.OdigosCollectorRoleLabel + "=" + string(k8sconsts.CollectorsRoleClusterGateway)})
	if err != nil {
		return nil, err
	}
	if len(depList.Items) == 0 {
		return nil, fmt.Errorf("no cluster gateway deployment found")
	}
	if len(depList.Items) > 1 {
		return nil, fmt.Errorf("multiple cluster gateway deployments found")
	}
	dep := &depList.Items[0]

	var hpa *autoscalingv2.HorizontalPodAutoscaler
	if h, err := kube.DefaultClient.AutoscalingV2().HorizontalPodAutoscalers(ns).Get(ctx, k8sconsts.OdigosClusterCollectorHpaName, metav1.GetOptions{}); err == nil {
		hpa = h
	} else {
		log.Printf("failed to get HPA %s/%s: %v", ns, k8sconsts.OdigosClusterCollectorHpaName, err)
	}

	result := &model.GatewayDeploymentInfo{}

	status, rolloutInProgress := computeDeploymentStatus(dep)
	result.Status = status
	result.RolloutInProgress = rolloutInProgress

	result.Hpa = computeGatewayHPA(dep, hpa)

	if rr := extractResourcesForContainer(dep.Spec.Template.Spec.Containers, k8sconsts.OdigosClusterCollectorContainerName); rr != nil {
		result.Resources = rr
	}

	result.ImageVersion = services.StringPtr(extractImageVersionForContainer(dep.Spec.Template.Spec.Containers, k8sconsts.OdigosClusterCollectorContainerName))

	result.LastRolloutAt = services.StringPtr(findLastRolloutTime(ctx, dep))

	manifestYAML, err := services.K8sDeploymentYamlManifest(dep)
	if err != nil {
		return nil, err
	}
	result.ManifestYaml = manifestYAML

	configMapYAML, err := services.K8sManifest(ctx, ns, model.K8sResourceKindConfigMap, k8sconsts.OdigosClusterCollectorConfigMapName)
	if err != nil {
		return nil, err
	}
	result.ConfigMapYaml = configMapYAML

	return result, nil
}

func computeDeploymentStatus(dep *appsv1.Deployment) (model.WorkloadRolloutStatus, bool) {
	var availableCond, progressingCond *appsv1.DeploymentCondition
	for i := range dep.Status.Conditions {
		c := dep.Status.Conditions[i]
		if c.Type == appsv1.DeploymentAvailable {
			availableCond = &c
		} else if c.Type == appsv1.DeploymentProgressing {
			progressingCond = &c
		}
	}

	if dep.Status.AvailableReplicas == 0 {
		return model.WorkloadRolloutStatusDown, dep.Status.UpdatedReplicas < dep.Status.Replicas || dep.Status.AvailableReplicas < dep.Status.Replicas
	}

	if availableCond == nil && progressingCond == nil {
		return model.WorkloadRolloutStatusUnknown, false
	}

	if progressingCond != nil {
		if progressingCond.Status == corev1.ConditionFalse || progressingCond.Reason == "ProgressDeadlineExceeded" {
			return model.WorkloadRolloutStatusFailed, false
		}
	}

	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	if progressingCond != nil && progressingCond.Status == corev1.ConditionTrue && dep.Status.UpdatedReplicas < desired {
		return model.WorkloadRolloutStatusUpdating, true
	}

	if availableCond != nil && availableCond.Status == corev1.ConditionFalse && progressingCond != nil && progressingCond.Status == corev1.ConditionTrue {
		return model.WorkloadRolloutStatusDegraded, dep.Status.UpdatedReplicas < dep.Status.Replicas || dep.Status.AvailableReplicas < dep.Status.Replicas
	}

	if availableCond != nil && availableCond.Status == corev1.ConditionTrue && progressingCond != nil && progressingCond.Status == corev1.ConditionTrue {
		if dep.Status.Replicas == dep.Status.UpdatedReplicas && dep.Status.Replicas == dep.Status.AvailableReplicas && dep.Status.Replicas == dep.Status.ReadyReplicas {
			return model.WorkloadRolloutStatusHealthy, false
		}
		return model.WorkloadRolloutStatusUpdating, true
	}

	return model.WorkloadRolloutStatusUnknown, false
}

func findLastRolloutTime(ctx context.Context, dep *appsv1.Deployment) string {
	if dep.Spec.Template.Annotations != nil {
		if v, ok := dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]; ok {
			return v
		}
	}
	if dep.Spec.Selector == nil {
		return ""
	}
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return ""
	}
	labelSelector := selector.String() + "," + k8sconsts.OdigosCollectorRoleLabel + "=" + string(k8sconsts.CollectorsRoleClusterGateway)
	rsList, err := kube.DefaultClient.AppsV1().ReplicaSets(dep.Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil || len(rsList.Items) == 0 {
		return ""
	}
	var latestOwned metav1.Time
	var latestAny metav1.Time
	for _, rs := range rsList.Items {
		if latestAny.IsZero() || rs.CreationTimestamp.After(latestAny.Time) {
			latestAny = rs.CreationTimestamp
		}
		for _, owner := range rs.OwnerReferences {
			if owner.Kind == "Deployment" && owner.UID == dep.UID {
				if latestOwned.IsZero() || rs.CreationTimestamp.After(latestOwned.Time) {
					latestOwned = rs.CreationTimestamp
				}
				break
			}
		}
	}
	if !latestOwned.IsZero() {
		return services.Metav1TimeToString(latestOwned)
	}
	if !latestAny.IsZero() {
		return services.Metav1TimeToString(latestAny)
	}
	return ""
}

func computeGatewayHPA(dep *appsv1.Deployment, hpa *autoscalingv2.HorizontalPodAutoscaler) *model.HorizontalPodAutoscalerInfo {
	if hpa == nil {
		return nil
	}
	h := &model.HorizontalPodAutoscalerInfo{}
	if hpa.Spec.MinReplicas != nil {
		v := int(*hpa.Spec.MinReplicas)
		h.Min = &v
	}
	max := int(hpa.Spec.MaxReplicas)
	h.Max = &max
	cur := int(dep.Status.Replicas)
	h.Current = &cur
	if hpa.Status.DesiredReplicas > 0 {
		d := int(hpa.Status.DesiredReplicas)
		h.Desired = &d
	} else if dep.Spec.Replicas != nil {
		d := int(*dep.Spec.Replicas)
		h.Desired = &d
	}

	if len(hpa.Status.Conditions) > 0 {
		for _, cond := range hpa.Status.Conditions {
			if cond.Type == autoscalingv2.ScalingActive {
				status := services.TransformConditionStatus(metav1.ConditionStatus(cond.Status), string(cond.Type), cond.Reason)
				lastTransitionTime := services.Metav1TimeToString(cond.LastTransitionTime)
				h.Conditions = append(h.Conditions, &model.Condition{
					Status:             status,
					Type:               string(cond.Type),
					Reason:             &cond.Reason,
					Message:            &cond.Message,
					LastTransitionTime: &lastTransitionTime,
				})
				break
			}
		}
	}
	return h
}
