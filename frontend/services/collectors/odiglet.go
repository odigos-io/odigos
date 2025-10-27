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
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "sigs.k8s.io/yaml"
)

func GetOdigletDaemonSetInfo(ctx context.Context) (*model.CollectorDaemonSetInfo, error) {
    ns := env.GetCurrentNamespace()
    name := k8sconsts.OdigletDaemonSetName

    ds, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
    if err != nil { return nil, err }

    status, inProgress := computeDaemonSetStatus(ds)
    nodes := &model.NodesSummary{ Desired: int(ds.Status.DesiredNumberScheduled), Ready: int(ds.Status.NumberReady) }

    result := &model.CollectorDaemonSetInfo{ Status: status, Nodes: nodes, RolloutInProgress: inProgress, ImageVersion: StringPtr(extractDaemonSetImageVersion(ds)), LastRolloutAt: StringPtr(findDaemonSetLastRolloutTime(ctx, ds)) }
    
	if rr := extractDaemonSetResources(ds); rr != nil { result.Resources = rr }
    
	return result, nil
}

func computeDaemonSetStatus(ds *appsv1.DaemonSet) (model.WorkloadStatus, bool) {
    desired := ds.Status.DesiredNumberScheduled
    updated := ds.Status.UpdatedNumberScheduled
    available := ds.Status.NumberAvailable
    ready := ds.Status.NumberReady
    unavailable := ds.Status.NumberUnavailable

    if available == 0 { return model.WorkloadStatusDown, updated < desired || available < desired }
    if updated < desired || available < desired { if unavailable > 0 { return model.WorkloadStatusDegraded, true }; return model.WorkloadStatusUpdating, true }
    if desired == updated && desired == available && desired == ready && unavailable == 0 { return model.WorkloadStatusHealthy, false }
    
	return model.WorkloadStatusUnknown, false
}

func extractDaemonSetResources(ds *appsv1.DaemonSet) *model.Resources {
    for _, c := range ds.Spec.Template.Spec.Containers { if c.Name == k8sconsts.OdigletContainerName {
        req := buildResourceAmounts(c.Resources.Requests)
        lim := buildResourceAmounts(c.Resources.Limits)
        if req == nil && lim == nil { return nil }
        return &model.Resources{ Requests: req, Limits: lim }
    }}
    return nil
}

func extractDaemonSetImageVersion(ds *appsv1.DaemonSet) string {
    for _, c := range ds.Spec.Template.Spec.Containers { if c.Name == k8sconsts.OdigletContainerName {
        image := c.Image; if idx := strings.Index(image, "@"); idx >= 0 { image = image[:idx] }
        parts := strings.Split(image, "/"); last := parts[len(parts)-1]
        if colon := strings.LastIndex(last, ":"); colon >= 0 { return last[colon+1:] }
        return last
    }}
    return ""
}

func findDaemonSetLastRolloutTime(ctx context.Context, ds *appsv1.DaemonSet) string {
    if ds.Spec.Template.Annotations != nil { if v, ok := ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]; ok { return v } }
    revList, err := kube.DefaultClient.AppsV1().ControllerRevisions(ds.Namespace).List(ctx, metav1.ListOptions{})
    if err == nil && len(revList.Items) > 0 {
        owned := make([]appsv1.ControllerRevision, 0, len(revList.Items))
        for _, cr := range revList.Items { for _, o := range cr.OwnerReferences { if o.Kind == "DaemonSet" && o.UID == ds.UID { owned = append(owned, cr); break } } }
        if len(owned) > 0 { sort.Slice(owned, func(i, j int) bool { return owned[i].CreationTimestamp.After(owned[j].CreationTimestamp.Time) }); return Metav1TimeToString(owned[0].CreationTimestamp) }
    }
    return Metav1TimeToString(ds.CreationTimestamp)
}


