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
	Applications []GetApplicationItem `json:"applications"`
}

type WorkloadKind string

const (
	WorkloadKindDeployment  WorkloadKind = "deployment"
	WorkloadKindStatefulSet WorkloadKind = "statefulset"
	WorkloadKindDaemonSet   WorkloadKind = "daemonset"
)

type GetApplicationItem struct {
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

	if IsSystemNamespace(request.Namespace) {
		// skip system namespaces which should not be instrumented
		c.JSON(http.StatusOK, GetApplicationsInNamespaceResponse{})
		return
	}

	ctx := c.Request.Context()
	deps, err := getDeployments(request.Namespace, ctx)
	if err != nil {
		returnError(c, err)
		return
	}

	ss, err := getStatefulSets(request.Namespace, ctx)
	if err != nil {
		returnError(c, err)
		return
	}

	dss, err := getDaemonSets(request.Namespace, ctx)
	if err != nil {
		returnError(c, err)
		return
	}

	items := make([]GetApplicationItem, len(deps)+len(ss)+len(dss))
	copy(items, deps)
	copy(items[len(deps):], ss)
	copy(items[len(deps)+len(ss):], dss)

	// check if the entire namespace is instrumented
	// as it affects the applications in the namespace
	// which use this label to determine if they should be instrumented
	namespace, err := kube.DefaultClient.CoreV1().Namespaces().Get(c.Request.Context(), request.Namespace, metav1.GetOptions{})
	if err != nil {
		returnError(c, err)
		return
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

	c.JSON(http.StatusOK, GetApplicationsInNamespaceResponse{
		Applications: items,
	})
}

func getDeployments(namespace string, ctx context.Context) ([]GetApplicationItem, error) {
	deps, err := kube.DefaultClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	response := make([]GetApplicationItem, len(deps.Items))
	for i, dep := range deps.Items {
		instrumentationLabel, found := dep.Labels[consts.OdigosInstrumentationLabel]
		var appInstrumentationLabeled *bool
		if found {
			instrumentationLabel := instrumentationLabel == consts.InstrumentationEnabled
			appInstrumentationLabeled = &instrumentationLabel
		}
		response[i] = GetApplicationItem{
			Name:                      dep.Name,
			Kind:                      WorkloadKindDeployment,
			Instances:                 int(dep.Status.AvailableReplicas),
			AppInstrumentationLabeled: appInstrumentationLabeled,
		}
	}

	return response, nil
}

func getStatefulSets(namespace string, ctx context.Context) ([]GetApplicationItem, error) {
	ss, err := kube.DefaultClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	response := make([]GetApplicationItem, len(ss.Items))
	for i, s := range ss.Items {
		response[i] = GetApplicationItem{
			Name:      s.Name,
			Kind:      WorkloadKindStatefulSet,
			Instances: int(s.Status.ReadyReplicas),
		}
	}

	return response, nil
}

func getDaemonSets(namespace string, ctx context.Context) ([]GetApplicationItem, error) {
	dss, err := kube.DefaultClient.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	response := make([]GetApplicationItem, len(dss.Items))
	for i, ds := range dss.Items {
		response[i] = GetApplicationItem{
			Name:      ds.Name,
			Kind:      WorkloadKindDaemonSet,
			Instances: int(ds.Status.NumberReady),
		}
	}

	return response, nil
}
