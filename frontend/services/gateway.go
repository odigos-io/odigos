package services

import (
    "context"
    "sort"
    "strings"

    "github.com/odigos-io/odigos/api/k8sconsts"
    "github.com/odigos-io/odigos/frontend/graph/model"
    "github.com/odigos-io/odigos/frontend/kube"
    "github.com/odigos-io/odigos/k8sutils/pkg/env"
    appsv1 "k8s.io/api/apps/v1"
    autoscalingv2 "k8s.io/api/autoscaling/v2"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/labels"
    "k8s.io/apimachinery/pkg/api/resource"
)

// GetGatewayDeploymentInfo fetches Deployment/HPA and computes the header info for the gateway deployment.
func GetGatewayDeploymentInfo(ctx context.Context) (*model.GatewayDeploymentInfo, error) {
    ns := env.GetCurrentNamespace()
    name := k8sconsts.OdigosClusterCollectorDeploymentName

    dep, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
    if err != nil {
        return nil, err
    }

    // Try to get HPA (v2). If not found, hpa will be nil and that's fine for UI.
    var hpa *autoscalingv2.HorizontalPodAutoscaler
    if h, err := kube.DefaultClient.AutoscalingV2().HorizontalPodAutoscalers(ns).Get(ctx, name, metav1.GetOptions{}); err == nil {
        hpa = h
    }

    result := &model.GatewayDeploymentInfo{}

    // Compute status and rollout
    status, rolloutInProgress := computeDeploymentStatus(dep)
    result.Status = status
    result.RolloutInProgress = rolloutInProgress

    // HPA fields
    if hpa != nil {
        h := &model.GatewayHPA{}
        if hpa.Spec.MinReplicas != nil {
            v := int(*hpa.Spec.MinReplicas)
            h.Min = &v
        }
        max := int(hpa.Spec.MaxReplicas)
        h.Max = &max
        cur := int(dep.Status.Replicas)
        h.Current = &cur
        // desired from HPA status if available, fallback to deployment spec replicas
        if hpa.Status.DesiredReplicas > 0 {
            d := int(hpa.Status.DesiredReplicas)
            h.Desired = &d
        } else if dep.Spec.Replicas != nil {
            d := int(*dep.Spec.Replicas)
            h.Desired = &d
        }
        result.Hpa = h
    }

    // Resources (requests/limits) for gateway container
    if rr := extractGatewayResources(dep); rr != nil {
        result.Resources = rr
    }

    // Image version
    result.ImageVersion = StringPtr(extractGatewayImageVersion(dep))

    // Last rollout timestamp
    result.LastRolloutAt = StringPtr(findLastRolloutTime(ctx, dep))

    return result, nil
}

func computeDeploymentStatus(dep *appsv1.Deployment) (model.GatewayDeploymentStatus, bool) {
    // Defaults
    var availableCond, progressingCond *metav1.Condition
    for i := range dep.Status.Conditions {
        c := dep.Status.Conditions[i]
        if c.Type == appsv1.DeploymentAvailable {
            availableCond = &c
        } else if c.Type == appsv1.DeploymentProgressing {
            progressingCond = &c
        }
    }

    // Down: no available replicas
    if dep.Status.AvailableReplicas == 0 {
        return model.GatewayDeploymentStatusDown, dep.Status.UpdatedReplicas < dep.Status.Replicas || dep.Status.AvailableReplicas < dep.Status.Replicas
    }

    // Unknown: no conditions
    if availableCond == nil && progressingCond == nil {
        return model.GatewayDeploymentStatusUnknown, false
    }

    // Failed: Progressing=False or reason ProgressDeadlineExceeded
    if progressingCond != nil {
        if progressingCond.Status == metav1.ConditionFalse || progressingCond.Reason == "ProgressDeadlineExceeded" {
            return model.GatewayDeploymentStatusFailed, false
        }
    }

    // Updating: progressing True but updatedReplicas < desiredReplicas
    desired := int32(1)
    if dep.Spec.Replicas != nil {
        desired = *dep.Spec.Replicas
    }
    if progressingCond != nil && progressingCond.Status == metav1.ConditionTrue && dep.Status.UpdatedReplicas < desired {
        return model.GatewayDeploymentStatusUpdating, true
    }

    // Degraded: Available=False but Progressing=True
    if availableCond != nil && availableCond.Status == metav1.ConditionFalse && progressingCond != nil && progressingCond.Status == metav1.ConditionTrue {
        return model.GatewayDeploymentStatusDegraded, dep.Status.UpdatedReplicas < dep.Status.Replicas || dep.Status.AvailableReplicas < dep.Status.Replicas
    }

    // Healthy: Available=True and Progressing=True and all replicas up to date
    if availableCond != nil && availableCond.Status == metav1.ConditionTrue && progressingCond != nil && progressingCond.Status == metav1.ConditionTrue {
        if dep.Status.Replicas == dep.Status.UpdatedReplicas && dep.Status.Replicas == dep.Status.AvailableReplicas && dep.Status.Replicas == dep.Status.ReadyReplicas {
            return model.GatewayDeploymentStatusHealthy, false
        }
        // otherwise still updating
        return model.GatewayDeploymentStatusUpdating, true
    }

    return model.GatewayDeploymentStatusUnknown, false
}

func extractGatewayResources(dep *appsv1.Deployment) *model.GatewayResources {
    for _, c := range dep.Spec.Template.Spec.Containers {
        if c.Name == k8sconsts.OdigosClusterCollectorContainerName {
            var req, lim *model.GatewayResourceAmounts
            if len(c.Resources.Requests) > 0 {
                req = &model.GatewayResourceAmounts{
                    CpuM:     int(c.Resources.Requests.Cpu().MilliValue()),
                    MemoryMiB: int(c.Resources.Requests.Memory().ScaledValue(resource.Mebi)),
                }
            }
            if len(c.Resources.Limits) > 0 {
                lim = &model.GatewayResourceAmounts{
                    CpuM:     int(c.Resources.Limits.Cpu().MilliValue()),
                    MemoryMiB: int(c.Resources.Limits.Memory().ScaledValue(resource.Mebi)),
                }
            }
            return &model.GatewayResources{Requests: req, Limits: lim}
        }
    }
    return nil
}

func extractGatewayImageVersion(dep *appsv1.Deployment) string {
    for _, c := range dep.Spec.Template.Spec.Containers {
        if c.Name == k8sconsts.OdigosClusterCollectorContainerName {
            image := c.Image
            if idx := strings.Index(image, "@"); idx >= 0 {
                image = image[:idx]
            }
            parts := strings.Split(image, "/")
            last := parts[len(parts)-1]
            if colon := strings.LastIndex(last, ":"); colon >= 0 {
                return last[colon+1:]
            }
            return last
        }
    }
    return ""
}

func findLastRolloutTime(ctx context.Context, dep *appsv1.Deployment) string {
    // Prefer restartedAt annotation on template
    if dep.Spec.Template.Annotations != nil {
        if v, ok := dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]; ok {
            return v
        }
    }

    // Else infer from newest ReplicaSet owned by this Deployment
    if dep.Spec.Selector == nil {
        return ""
    }
    selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
    if err != nil {
        return ""
    }
    rsList, err := kube.DefaultClient.AppsV1().ReplicaSets(dep.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
    if err != nil || len(rsList.Items) == 0 {
        return ""
    }

    // Filter owned by deployment
    owned := make([]appsv1.ReplicaSet, 0, len(rsList.Items))
    for _, rs := range rsList.Items {
        for _, owner := range rs.OwnerReferences {
            if owner.Kind == "Deployment" && owner.UID == dep.UID {
                owned = append(owned, rs)
                break
            }
        }
    }
    if len(owned) == 0 {
        owned = rsList.Items
    }
    sort.Slice(owned, func(i, j int) bool { return owned[i].CreationTimestamp.After(owned[j].CreationTimestamp.Time) })
    return Metav1TimeToString(owned[0].CreationTimestamp)
}


