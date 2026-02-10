package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetK8SNamespaces(ctx context.Context, namespaceName *string) ([]*model.K8sActualNamespace, error) {
	var namespaces []corev1.Namespace
	var response []*model.K8sActualNamespace

	if namespaceName == nil || *namespaceName == "" {
		relevantNameSpaces, err := getRelevantNameSpaces(ctx)
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

	var allNsSources v1alpha1.SourceList
	if err := kube.CacheClient.List(ctx, &allNsSources, ctrlclient.MatchingLabels{
		k8sconsts.WorkloadKindLabel: string(k8sconsts.WorkloadKindNamespace),
	}); err != nil {
		return nil, err
	}
	nsSourceMap := make(map[string]*v1alpha1.Source)
	for i := range allNsSources.Items {
		s := &allNsSources.Items[i]
		nsSourceMap[s.Spec.Workload.Name] = s
	}

	for _, item := range namespaces {
		nsName := item.Name

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
func getRelevantNameSpaces(ctx context.Context) ([]corev1.Namespace, error) {
	odigosConfiguration, err := GetOdigosConfiguration(ctx)
	if err != nil {
		return nil, err
	}

	list, err := kube.DefaultClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]corev1.Namespace, 0, len(list.Items))
	for _, namespace := range list.Items {
		if utils.IsItemIgnored(namespace.Name, odigosConfiguration.IgnoredNamespaces) {
			continue
		}
		result = append(result, namespace)
	}

	return result, nil
}

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
