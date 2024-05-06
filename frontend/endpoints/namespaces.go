package endpoints

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/multierr"

	"github.com/odigos-io/odigos/common/consts"

	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gin-gonic/gin"
)

type GetNamespacesResponse struct {
	Namespaces []GetNamespaceItem `json:"namespaces"`
}

type GetNamespaceItem struct {
	Name      string `json:"name"`
	Selected  bool   `json:"selected"`
	TotalApps int    `json:"totalApps"`
}

func GetNamespaces(c *gin.Context) {
	list, err := kube.DefaultClient.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	appsPerNamespace, err := CountAppsPerNamespace(c)
	if err != nil {
		returnError(c, err)
		return
	}

	var response GetNamespacesResponse
	for _, namespace := range list.Items {

		if IsSystemNamespace(namespace.Name) {
			// skip system namespaces which should not be instrumented
			continue
		}

		// check if entire namespace is instrumented
		selected := namespace.Labels[consts.OdigosInstrumentationLabel] == consts.InstrumentationEnabled

		response.Namespaces = append(response.Namespaces, GetNamespaceItem{
			Name:      namespace.Name,
			Selected:  selected,
			TotalApps: appsPerNamespace[namespace.Name],
		})
	}

	c.JSON(http.StatusOK, response)
}

type PersistNamespaceItem struct {
	Name           string                   `json:"name"`
	SelectedAll    bool                     `json:"selected_all"`
	FutureSelected *bool                    `json:"future_selected,omitempty"`
	Objects        []PersistNamespaceObject `json:"objects"`
}

type PersistNamespaceObject struct {
	Name     string       `json:"name"`
	Kind     WorkloadKind `json:"kind"`
	Selected *bool        `json:"selected,omitempty"`
}

func PersistNamespaces(c *gin.Context) {
	request := make(map[string]PersistNamespaceItem)
	if err := c.ShouldBindJSON(&request); err != nil {
		returnError(c, err)
		return
	}

	for nsName, nsItem := range request {
		if IsSystemNamespace(nsName) {
			// skip system namespaces which should not be instrumented
			continue
		}

		jsonMergePayload := getJsonMergePatchForInstrumentationLabel(nsItem.FutureSelected)
		_, err := kube.DefaultClient.CoreV1().Namespaces().Patch(c.Request.Context(), nsName, types.MergePatchType, jsonMergePayload, metav1.PatchOptions{})
		if err != nil {
			returnError(c, err)
			return
		}

		err = syncWorkloadsInNamespace(c.Request.Context(), nsName, nsItem.Objects)
		if err != nil {
			returnError(c, err)
			return
		}
	}
}

func getJsonMergePatchForInstrumentationLabel(enabled *bool) []byte {
	labelJsonMergePatchValue := "null"
	if enabled != nil {
		if *enabled {
			labelJsonMergePatchValue = fmt.Sprintf("\"%s\"", consts.InstrumentationEnabled)
		} else {
			labelJsonMergePatchValue = fmt.Sprintf("\"%s\"", consts.InstrumentationDisabled)
		}
	}

	jsonMergePatchContent := fmt.Sprintf(`{"metadata":{"labels":{"%s":%s}}}`, consts.OdigosInstrumentationLabel, labelJsonMergePatchValue)
	return []byte(jsonMergePatchContent)
}
func syncWorkloadsInNamespace(ctx context.Context, nsName string, workloads []PersistNamespaceObject) error {
	var errs error
	for _, workload := range workloads {
		err := setWorkloadInstrumentationLabel(ctx, nsName, workload.Name, workload.Kind, workload.Selected)
		errs = multierr.Append(errs, err)
	}
	return errs
}

// returns a map, where the key is a namespace name and the value is the
// number of apps in this namespace (not necessarily instrumented)
func CountAppsPerNamespace(ctx context.Context) (map[string]int, error) {

	deps, err := kube.DefaultClient.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ss, err := kube.DefaultClient.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ds, err := kube.DefaultClient.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaceToAppsCount := make(map[string]int)
	for _, dep := range deps.Items {
		namespaceToAppsCount[dep.Namespace]++
	}
	for _, st := range ss.Items {
		namespaceToAppsCount[st.Namespace]++
	}
	for _, d := range ds.Items {
		namespaceToAppsCount[d.Namespace]++
	}

	return namespaceToAppsCount, nil
}
