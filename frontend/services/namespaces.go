package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"golang.org/x/sync/errgroup"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/yaml"
)

type GetNamespacesResponse struct {
	Namespaces []model.K8sActualNamespace `json:"namespaces"`
}

func GetK8SNamespaces(ctx context.Context) (GetNamespacesResponse, error) {
	relevantNameSpaces, err := getRelevantNameSpaces(ctx, consts.DefaultOdigosNamespace)
	if err != nil {
		return GetNamespacesResponse{}, err
	}

	var response GetNamespacesResponse
	for _, namespace := range relevantNameSpaces {
		nsName := namespace.Name

		// check if entire namespace is instrumented
		source, err := GetSourceCRD(ctx, nsName, nsName, WorkloadKindNamespace)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return GetNamespacesResponse{}, err
		}

		selected := source != nil
		response.Namespaces = append(response.Namespaces, model.K8sActualNamespace{
			Name:     nsName,
			Selected: selected,
		})
	}

	return response, nil
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
		configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(odigosns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfig); err != nil {
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
		}).List, ctx, &metav1.ListOptions{}, func(list *metav1.PartialObjectMetadataList) error {
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

func SyncWorkloadsInNamespace(ctx context.Context, nsName string, workloads []model.PersistNamespaceSourceInput) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(kube.K8sClientDefaultBurst)

	for _, workload := range workloads {
		g.Go(func() error {
			return ToggleSourceCRD(ctx, nsName, workload.Name, WorkloadKind(workload.Kind.String()), workload.Selected)
		})
	}

	return g.Wait()
}
