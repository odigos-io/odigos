package endpoints

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GetApplicationsInNamespaceRequest struct {
	Namespace string `uri:"namespace" binding:"required"`
}

type GetApplicationsInNamespaceResponse struct {
	Applications []GetApplicationItem `json:"applications"`
}

type ApplicationKind string

const (
	ApplicationKindDeployment  ApplicationKind = "deployment"
	ApplicationKindStatefulSet ApplicationKind = "statefulset"
	ApplicationKindDaemonSet   ApplicationKind = "daemonset"
)

type GetApplicationItem struct {
	Name      string          `json:"name"`
	Kind      ApplicationKind `json:"kind"`
	Instances int             `json:"instances"`
	Selected  bool            `json:"selected"`
}

func GetApplicationsInNamespace(c *gin.Context) {
	var request GetApplicationsInNamespaceRequest
	if err := c.ShouldBindUri(&request); err != nil {
		returnError(c, err)
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
		response[i] = GetApplicationItem{
			Name:      dep.Name,
			Kind:      ApplicationKindDeployment,
			Instances: int(dep.Status.AvailableReplicas),
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
			Kind:      ApplicationKindStatefulSet,
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
			Kind:      ApplicationKindDaemonSet,
			Instances: int(ds.Status.NumberReady),
		}
	}

	return response, nil
}
