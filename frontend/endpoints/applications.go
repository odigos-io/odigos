package endpoints

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GetApplicationsInNamespaceRequest struct {
	Namespace string `uri:"namespace" binding:"required"`
}

type GetApplicationsInNamespaceResponse struct {
	Applications []GetApplicationItemInNamespace `json:"applications"`
}

type WorkloadKind string

const (
	WorkloadKindDeployment  WorkloadKind = "Deployment"
	WorkloadKindStatefulSet WorkloadKind = "StatefulSet"
	WorkloadKindDaemonSet   WorkloadKind = "DaemonSet"
)

type GetApplicationItemInNamespace struct {
	Name                      string       `json:"name"`
	Kind                      WorkloadKind `json:"kind"`
	Instances                 int          `json:"instances"`
	AppInstrumentationLabeled *bool        `json:"app_instrumentation_labeled"`
	NsInstrumentationLabeled  *bool        `json:"ns_instrumentation_labeled"`
	InstrumentationEffective  bool         `json:"instrumentation_effective"`
}

func GetApplicationsInNamespace(c *gin.Context) {
	var request GetApplicationsInNamespaceRequest
	if err := c.ShouldBindUri(&request); err != nil {
		returnError(c, err)
		return
	}

	ctx := c.Request.Context()
	items, err := getApplicationsInNamespace(ctx, request.Namespace)
	if err != nil {
		returnError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetApplicationsInNamespaceResponse{
		Applications: items,
	})
}

func getApplicationsInNamespace(ctx context.Context, ns string) ([]GetApplicationItemInNamespace, error) {
	deps, err := getDeployments(ns, ctx)
	if err != nil {
		return nil, err
	}

	ss, err := getStatefulSets(ns, ctx)
	if err != nil {
		return nil, err
	}

	dss, err := getDaemonSets(ns, ctx)
	if err != nil {
		return nil, err
	}

	items := make([]GetApplicationItemInNamespace, len(deps)+len(ss)+len(dss))
	copy(items, deps)
	copy(items[len(deps):], ss)
	copy(items[len(deps)+len(ss):], dss)

	// check if the entire namespace is instrumented
	// as it affects the applications in the namespace
	// which use this label to determine if they should be instrumented
	namespace, err := kube.DefaultClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	namespaceInstrumented, found := namespace.Labels[consts.OdigosInstrumentationLabel]
	var nsInstrumentationLabeled *bool
	if found {
		instrumentationLabel := namespaceInstrumented == consts.InstrumentationEnabled
		nsInstrumentationLabeled = &instrumentationLabel
	}
	for i := range items {
		item := &items[i]
		item.NsInstrumentationLabeled = nsInstrumentationLabeled
		appInstrumented := (item.AppInstrumentationLabeled != nil && *item.AppInstrumentationLabeled)
		appInstrumentationInherited := item.AppInstrumentationLabeled == nil
		nsInstrumented := (nsInstrumentationLabeled != nil && *nsInstrumentationLabeled)
		item.InstrumentationEffective = appInstrumented || (appInstrumentationInherited && nsInstrumented)
	}

	return items, nil
}

func getDeployments(namespace string, ctx context.Context) ([]GetApplicationItemInNamespace, error) {
	deps, err := kube.DefaultClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	response := make([]GetApplicationItemInNamespace, len(deps.Items))
	for i, dep := range deps.Items {
		instrumentationLabel, found := dep.Labels[consts.OdigosInstrumentationLabel]
		var appInstrumentationLabeled *bool
		if found {
			instrumentationLabel := instrumentationLabel == consts.InstrumentationEnabled
			appInstrumentationLabeled = &instrumentationLabel
		}
		response[i] = GetApplicationItemInNamespace{
			Name:                      dep.Name,
			Kind:                      WorkloadKindDeployment,
			Instances:                 int(dep.Status.AvailableReplicas),
			AppInstrumentationLabeled: appInstrumentationLabeled,
		}
	}

	return response, nil
}

func getStatefulSets(namespace string, ctx context.Context) ([]GetApplicationItemInNamespace, error) {
	ss, err := kube.DefaultClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	response := make([]GetApplicationItemInNamespace, len(ss.Items))
	for i, s := range ss.Items {
		response[i] = GetApplicationItemInNamespace{
			Name:      s.Name,
			Kind:      WorkloadKindStatefulSet,
			Instances: int(s.Status.ReadyReplicas),
		}
	}

	return response, nil
}

func getDaemonSets(namespace string, ctx context.Context) ([]GetApplicationItemInNamespace, error) {
	dss, err := kube.DefaultClient.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	response := make([]GetApplicationItemInNamespace, len(dss.Items))
	for i, ds := range dss.Items {
		response[i] = GetApplicationItemInNamespace{
			Name:      ds.Name,
			Kind:      WorkloadKindDaemonSet,
			Instances: int(ds.Status.NumberReady),
		}
	}

	return response, nil
}
