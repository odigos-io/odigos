package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"
)

func GetK8SNamespaces(ctx context.Context, namespaceName *string) ([]*model.K8sActualNamespace, error) {
	var namespaces []corev1.Namespace
	var response []*model.K8sActualNamespace

	if namespaceName == nil || *namespaceName == "" {
		relevantNameSpaces, err := getRelevantNameSpaces(ctx, env.GetCurrentNamespace())
		if err != nil {
			return nil, err
		}
		namespaces = relevantNameSpaces
	} else {
		namespace, err := kube.DefaultClient.CoreV1().Namespaces().Get(ctx, *namespaceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		namespaces = []corev1.Namespace{*namespace}
	}

	// Batch-fetch all namespace-kind sources in one call instead of per-namespace
	allNsSources, err := kube.DefaultClient.OdigosClient.Sources("").List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			k8sconsts.WorkloadKindLabel: string(k8sconsts.WorkloadKindNamespace),
		}).String(),
	})
	if err != nil {
		return nil, err
	}
	nsSourceMap := make(map[string]*v1alpha1.Source)
	for i := range allNsSources.Items {
		s := &allNsSources.Items[i]
		nsSourceMap[s.Spec.Workload.Name] = s
	}

	for _, item := range namespaces {
		nsName := item.Name

		// check if entire namespace is instrumented via map lookup
		source := nsSourceMap[nsName]

		instrumented := source != nil && !source.Spec.DisableInstrumentation
		response = append(response, &model.K8sActualNamespace{
			Name:            nsName,
			DataStreamNames: ExtractDataStreamsFromSource(source, nil),
			Selected:        instrumented,
		})
	}

	return response, nil
}

// getRelevantNameSpaces returns a list of namespaces that are relevant for instrumentation.
// Taking into account the ignored namespaces from the OdigosConfiguration.
func getRelevantNameSpaces(ctx context.Context, odigosns string) ([]corev1.Namespace, error) {
	var (
		odigosConfiguration *common.OdigosConfiguration
		list                *corev1.NamespaceList
	)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(odigosns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration); err != nil {
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
		return []corev1.Namespace{}, err
	}

	result := []corev1.Namespace{}
	for _, namespace := range list.Items {
		if utils.IsItemIgnored(namespace.Name, odigosConfiguration.IgnoredNamespaces) {
			continue
		}

		result = append(result, namespace)
	}

	return result, nil
}

// returns a map, where the key is a namespace name and the value is the
// number of apps in this namespace (not necessarily instrumented).
// Uses the in-memory cache for instant reads instead of API calls.
func CountAppsPerNamespace(ctx context.Context) (map[string]int, error) {
	counts := make(map[string]int)

	var deps appsv1.DeploymentList
	if err := kube.CacheClient.List(ctx, &deps); err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	for _, d := range deps.Items {
		counts[d.Namespace]++
	}

	var stss appsv1.StatefulSetList
	if err := kube.CacheClient.List(ctx, &stss); err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}
	for _, s := range stss.Items {
		counts[s.Namespace]++
	}

	var dss appsv1.DaemonSetList
	if err := kube.CacheClient.List(ctx, &dss); err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}
	for _, d := range dss.Items {
		counts[d.Namespace]++
	}

	return counts, nil
}

func SyncWorkloadsInNamespace(ctx context.Context, workloads []*model.PersistNamespaceSourceInput) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(k8sconsts.K8sClientDefaultBurst)

	for _, workload := range workloads {
		g.Go(func() error {
			return ToggleSourceCRD(ctx, workload.Namespace, workload.Name, workload.Kind, workload.Selected, workload.CurrentStreamName)
		})
	}

	return g.Wait()
}
