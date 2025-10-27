package services

import (
    "context"
    "encoding/json"
    "sort"
    "strings"

    "github.com/odigos-io/odigos/api/k8sconsts"
    "github.com/odigos-io/odigos/frontend/graph/model"
    "github.com/odigos-io/odigos/frontend/kube"
    "github.com/odigos-io/odigos/k8sutils/pkg/env"
    appsv1 "k8s.io/api/apps/v1"
    autoscalingv2 "k8s.io/api/autoscaling/v2"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "sigs.k8s.io/yaml"
)

// GetGatewayDeploymentInfo fetches Deployment/HPA and computes the header info for the gateway deployment.
func GetGatewayDeploymentInfo(ctx context.Context) (*model.GatewayDeploymentInfo, error) {
    ns := env.GetCurrentNamespace()
    name := k8sconsts.OdigosClusterCollectorDeploymentName

    dep, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
    if err != nil {
        return nil, err
    }

    var hpa *autoscalingv2.HorizontalPodAutoscaler
    if h, err := kube.DefaultClient.AutoscalingV2().HorizontalPodAutoscalers(ns).Get(ctx, name, metav1.GetOptions{}); err == nil {
        hpa = h
    }

    result := &model.GatewayDeploymentInfo{}

    
    status, rolloutInProgress := computeDeploymentStatus(dep)
    result.Status = status
    result.RolloutInProgress = rolloutInProgress

    result.Hpa = computeGatewayHPA(dep, hpa)

    if rr := extractGatewayResources(dep); rr != nil {
        result.Resources = rr
    }

    result.ImageVersion = StringPtr(extractGatewayImageVersion(dep))

    result.LastRolloutAt = StringPtr(findLastRolloutTime(ctx, dep))

    return result, nil
}

func computeDeploymentStatus(dep *appsv1.Deployment) (model.WorkloadStatus, bool) {
    var availableCond, progressingCond *appsv1.DeploymentCondition
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
        return model.WorkloadStatusDown, dep.Status.UpdatedReplicas < dep.Status.Replicas || dep.Status.AvailableReplicas < dep.Status.Replicas
    }

    // Unknown: no conditions
    if availableCond == nil && progressingCond == nil {
        return model.WorkloadStatusUnknown, false
    }

    // Failed: Progressing=False or reason ProgressDeadlineExceeded
    if progressingCond != nil {
        if progressingCond.Status == corev1.ConditionFalse || progressingCond.Reason == "ProgressDeadlineExceeded" {
            return model.WorkloadStatusFailed, false
        }
    }

    // Updating: progressing True but updatedReplicas < desiredReplicas
    desired := int32(1)
    if dep.Spec.Replicas != nil {
        desired = *dep.Spec.Replicas
    }
    if progressingCond != nil && progressingCond.Status == corev1.ConditionTrue && dep.Status.UpdatedReplicas < desired {
        return model.WorkloadStatusUpdating, true
    }

    // Degraded: Available=False but Progressing=True
    if availableCond != nil && availableCond.Status == corev1.ConditionFalse && progressingCond != nil && progressingCond.Status == corev1.ConditionTrue {
        return model.WorkloadStatusDegraded, dep.Status.UpdatedReplicas < dep.Status.Replicas || dep.Status.AvailableReplicas < dep.Status.Replicas
    }

    // Healthy: Available=True and Progressing=True and all replicas up to date
    if availableCond != nil && availableCond.Status == corev1.ConditionTrue && progressingCond != nil && progressingCond.Status == corev1.ConditionTrue {
        if dep.Status.Replicas == dep.Status.UpdatedReplicas && dep.Status.Replicas == dep.Status.AvailableReplicas && dep.Status.Replicas == dep.Status.ReadyReplicas {
            return model.WorkloadStatusHealthy, false
        }

        return model.WorkloadStatusUpdating, true
    }

    return model.WorkloadStatusUnknown, false
}

func extractGatewayResources(dep *appsv1.Deployment) *model.Resources {
    for _, c := range dep.Spec.Template.Spec.Containers {
        if c.Name == k8sconsts.OdigosClusterCollectorContainerName {
            var req, lim *model.ResourceAmounts
            if len(c.Resources.Requests) > 0 {
                memBytes := c.Resources.Requests.Memory().Value()
                req = &model.ResourceAmounts{
                    CPUM:      int(c.Resources.Requests.Cpu().MilliValue()),
                    MemoryMiB: int(memBytes / (1024 * 1024)),
                }
            }
            if len(c.Resources.Limits) > 0 {
                memBytes := c.Resources.Limits.Memory().Value()
                lim = &model.ResourceAmounts{
                    CPUM:      int(c.Resources.Limits.Cpu().MilliValue()),
                    MemoryMiB: int(memBytes / (1024 * 1024)),
                }
            }
            return &model.Resources{Requests: req, Limits: lim}
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
    rsList, err := kube.DefaultClient.AppsV1().ReplicaSets(dep.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
    if err != nil || len(rsList.Items) == 0 {
        return ""
    }

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
    return h
}




