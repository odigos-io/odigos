package endpoints

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/utils"

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

func GetNamespaces(c *gin.Context, odigosns string) {
	response, err := getRelevantNameSpaces(c.Request.Context(), odigosns)
	if err != nil {
		returnError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// getRelevantNameSpaces returns a list of namespaces that are relevant for instrumentation.
// Taking into account the ignored namespaces from the OdigosConfiguration.
func getRelevantNameSpaces(ctx context.Context, odigosns string) (GetNamespacesResponse, error) {
	odigosConfig, err := kube.DefaultClient.OdigosClient.OdigosConfigurations(odigosns).Get(ctx, consts.DefaultOdigosConfigurationName, metav1.GetOptions{})
	if err != nil {
		return GetNamespacesResponse{}, err
	}

	list, err := kube.DefaultClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return GetNamespacesResponse{}, err
	}

	appsPerNamespace, err := CountAppsPerNamespace(ctx)
	if err != nil {
		return GetNamespacesResponse{}, err
	}

	var response GetNamespacesResponse
	for _, namespace := range list.Items {
		if utils.IsNamespaceIgnored(namespace.Name, odigosConfig.Spec.IgnoredNamespaces) {
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

	return response, nil
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
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(kube.K8sClientDefaultBurst)

	for _, workload := range workloads {
		currWorkload := workload
		g.Go(func() error {
			return setWorkloadInstrumentationLabel(ctx, nsName, currWorkload.Name, currWorkload.Kind, currWorkload.Selected)
		})
	}
	return g.Wait()
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
