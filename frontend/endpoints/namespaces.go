package endpoints

import (
	"context"
	"fmt"
	"net/http"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"golang.org/x/sync/errgroup"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/utils"

	"github.com/odigos-io/odigos/frontend/kube"
	v1 "k8s.io/api/core/v1"
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
	ctx := c.Request.Context()
	var (
		relevantNameSpaces []v1.Namespace
		appsPerNamespace   map[string]int
	)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		relevantNameSpaces, err = getRelevantNameSpaces(ctx, odigosns)
		return err
	})

	g.Go(func() error {
		var err error
		appsPerNamespace, err = CountAppsPerNamespace(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		returnError(c, err)
		return
	}

	var response GetNamespacesResponse
	for _, namespace := range relevantNameSpaces {
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

// getRelevantNameSpaces returns a list of namespaces that are relevant for instrumentation.
// Taking into account the ignored namespaces from the OdigosConfiguration.
func getRelevantNameSpaces(ctx context.Context, odigosns string) ([]v1.Namespace, error) {
	var (
		odigosConfig *common.OdigosConfiguration
		list         *v1.NamespaceList
	)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(odigosns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), odigosConfig); err != nil {
			return err
		}
		return err
	})

	g.Go(func() error {
		var err error
		list, err = kube.DefaultClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		return err
	})

	if err := g.Wait(); err != nil {
		return []v1.Namespace{}, err
	}

	result := []v1.Namespace{}
	for _, namespace := range list.Items {
		if utils.IsItemIgnored(namespace.Name, odigosConfig.IgnoredNamespaces) {
			continue
		}

		result = append(result, namespace)
	}

	return result, nil
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
			// Only label selected sources, ignore the rest
			if currWorkload.Selected != nil && *currWorkload.Selected {
				return setWorkloadInstrumentationLabel(ctx, nsName, currWorkload.Name, currWorkload.Kind, currWorkload.Selected)
			}
			return nil
		})
	}
	return g.Wait()
}

// returns a map, where the key is a namespace name and the value is the
// number of apps in this namespace (not necessarily instrumented)
func CountAppsPerNamespace(ctx context.Context) (map[string]int, error) {
	namespaceToAppsCount := make(map[string]int)

	resourceTypes := []string{"deployments", "statefulsets", "daemonsets"}

	for _, resourceType := range resourceTypes {
		err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.MetadataClient.Resource(schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: resourceType,
		}).List, ctx, metav1.ListOptions{}, func(list *metav1.PartialObjectMetadataList) error {
			for _, item := range list.Items {
				namespaceToAppsCount[item.Namespace]++
			}
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to count %s: %w", resourceType, err)
		}
	}

	return namespaceToAppsCount, nil
}
