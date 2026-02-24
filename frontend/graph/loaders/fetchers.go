package loaders

import (
	"context"
	"fmt"
	"time"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// formatOperationMessage creates a clean operation message that handles empty values gracefully
func formatOperationMessage(operation string, namespace string, additionalInfo ...string) string {
	if namespace == "" {
		namespace = "all namespaces"
	}

	if len(additionalInfo) > 0 && additionalInfo[0] != "" {
		return fmt.Sprintf("%s in namespace %s with %s", operation, namespace, additionalInfo[0])
	}
	return fmt.Sprintf("%s in namespace %s", operation, namespace)
}

// timedAPICall wraps a Kubernetes API call with timing and logging
func timedAPICall[T any](logger logr.Logger, operation string, apiCall func() (T, error)) (T, error) {
	start := time.Now()
	result, err := apiCall()
	duration := time.Since(start)

	if err != nil {
		logger.Error(err, "API call failed", "operation", operation, "duration", duration)
	} else {
		logger.Info("API call completed", "operation", operation, "duration", duration)
	}

	return result, err
}

// function to get just the instrumentation configs that match the filter.
// e.g. load only sources which are marked for instrumentation after the instrumentor reconciles it.
// this is cheaper and faster query than to load all the sources and resolve each one.
func fetchInstrumentationConfigs(ctx context.Context, logger logr.Logger, filters *WorkloadFilter, k8sCacheClient client.Client) (map[model.K8sWorkloadID]*odigosv1.InstrumentationConfig, error) {

	// diffrentiate between a single source query and a namespace / cluster wide query.
	if filters.SingleWorkload != nil {
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(filters.SingleWorkload.WorkloadName, filters.SingleWorkload.WorkloadKind)
		var instrumentationConfig odigosv1.InstrumentationConfig
		err := k8sCacheClient.Get(ctx, client.ObjectKey{
			Namespace: filters.NamespaceString,
			Name:      instrumentationConfigName,
		}, &instrumentationConfig)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// workload cam be not found and it is not an error.
				// we will just skip it.
				return nil, nil
			}
			return nil, err
		}
		return map[model.K8sWorkloadID]*odigosv1.InstrumentationConfig{
			{
				Namespace: instrumentationConfig.Namespace,
				Kind:      model.K8sResourceKind(filters.SingleWorkload.WorkloadKind),
				Name:      filters.SingleWorkload.WorkloadName,
			}: &instrumentationConfig,
		}, nil
	} else {
		var instrumentationConfigs odigosv1.InstrumentationConfigList
		err := k8sCacheClient.List(ctx, &instrumentationConfigs, client.InNamespace(filters.NamespaceString))
		if err != nil {
			return nil, err
		}
		configById := make(map[model.K8sWorkloadID]*odigosv1.InstrumentationConfig, len(instrumentationConfigs.Items))
		for _, config := range instrumentationConfigs.Items {
			if _, ok := filters.IgnoredNamespaces[config.Namespace]; ok {
				continue
			}
			pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(config.Name, config.Namespace)
			if err != nil {
				return nil, err
			}
			sourceId := model.K8sWorkloadID{
				Namespace: config.Namespace,
				Kind:      model.K8sResourceKind(pw.Kind),
				Name:      pw.Name,
			}
			configById[sourceId] = &config
		}
		return configById, nil
	}
}

func fetchSourcesForWorkload(ctx context.Context, filters *WorkloadFilterSingleWorkload, k8sCacheClient client.Client) (*odigosv1.SourceList, error) {
	// for workload we need to fetch both the workload and namespace sources.
	workloadLabels := map[string]string{
		k8sconsts.WorkloadNamespaceLabel: filters.Namespace,
		k8sconsts.WorkloadKindLabel:      string(filters.WorkloadKind),
		k8sconsts.WorkloadNameLabel:      filters.WorkloadName,
	}
	workloadSources := &odigosv1.SourceList{}
	err := k8sCacheClient.List(ctx, workloadSources, client.InNamespace(filters.Namespace), client.MatchingLabels(workloadLabels))
	if err != nil {
		return nil, err
	}

	namespaceLabels := map[string]string{
		k8sconsts.WorkloadNamespaceLabel: filters.Namespace,
		k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
		k8sconsts.WorkloadNameLabel:      filters.Namespace,
	}
	namespaceSources := &odigosv1.SourceList{}
	err = k8sCacheClient.List(ctx, namespaceSources, client.InNamespace(filters.Namespace), client.MatchingLabels(namespaceLabels))
	if err != nil {
		return nil, err
	}

	// merge the two lists into a odigosv1.SourceList
	allSources := &odigosv1.SourceList{
		Items: append(workloadSources.Items, namespaceSources.Items...),
	}

	return allSources, nil
}

func fetchSourcesForNamespace(ctx context.Context, filters *WorkloadFilterSingleNamespace, k8sCacheClient client.Client) (*odigosv1.SourceList, error) {
	labels := map[string]string{
		k8sconsts.WorkloadNamespaceLabel: filters.Namespace,
	}
	// will return both "workload" sources and "namespace" sources as required
	sources := &odigosv1.SourceList{}
	// assumes that sources are in the same namespace they are instrumenting (which is true at time of writing)
	err := k8sCacheClient.List(ctx, sources, client.InNamespace(filters.Namespace), client.MatchingLabels(labels))
	if err != nil {
		return nil, err
	}
	return sources, nil
}

func fetchAllSources(ctx context.Context, ignoredNamespaces map[string]struct{}, k8sCacheClient client.Client) (*odigosv1.SourceList, error) {
	sources := &odigosv1.SourceList{}
	err := k8sCacheClient.List(ctx, sources, client.MatchingLabels(map[string]string{}))
	if err != nil {
		return nil, err
	}

	filteredSources := make([]odigosv1.Source, 0, len(sources.Items))
	for _, source := range sources.Items {
		if _, ok := ignoredNamespaces[source.Namespace]; ok {
			continue
		}
		filteredSources = append(filteredSources, source)
	}
	sources.Items = filteredSources
	return sources, nil
}

func fetchSources(ctx context.Context, filters *WorkloadFilter, k8sCacheClient client.Client) (workloadSources map[model.K8sWorkloadID]*odigosv1.Source, namespaceSources map[string]*odigosv1.Source, err error) {

	var sources *odigosv1.SourceList
	if filters.SingleWorkload != nil {
		sources, err = fetchSourcesForWorkload(ctx, filters.SingleWorkload, k8sCacheClient)
	} else if filters.SingleNamespace != nil {
		sources, err = fetchSourcesForNamespace(ctx, filters.SingleNamespace, k8sCacheClient)
	} else {
		sources, err = fetchAllSources(ctx, filters.IgnoredNamespaces, k8sCacheClient)
	}
	if err != nil {
		return nil, nil, err
	}

	workloadSources = make(map[model.K8sWorkloadID]*odigosv1.Source, len(sources.Items)) // assuming most source are workload so len is almost right
	namespaceSources = make(map[string]*odigosv1.Source)                                 // expecting only few of these
	for _, source := range sources.Items {
		wd := source.Spec.Workload
		sourceId := model.K8sWorkloadID{
			Namespace: wd.Namespace,
			Kind:      model.K8sResourceKind(wd.Kind),
			Name:      wd.Name,
		}
		if wd.Kind == k8sconsts.WorkloadKindNamespace {
			namespaceSources[wd.Name] = &source
		} else {
			workloadSources[sourceId] = &source
		}
	}
	return
}

func fetchWorkloadManifests(ctx context.Context, logger logr.Logger, filters *WorkloadFilter, k8sCacheClient client.Client) (workloadManifests map[model.K8sWorkloadID]*computed.CachedWorkloadManifest, err error) {

	// if this is a query for one specific workload, then fetch only it.
	if filters.SingleWorkload != nil {
		workloadManifests = make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
		switch filters.SingleWorkload.WorkloadKind {
		case k8sconsts.WorkloadKindDeployment:
			deployment := &appsv1.Deployment{}
			err := k8sCacheClient.Get(ctx, client.ObjectKey{
				Namespace: filters.NamespaceString,
				Name:      filters.SingleWorkload.WorkloadName,
			}, deployment)
			if err != nil {
				return nil, client.IgnoreNotFound(err)
			}
			workloadHealthStatus := status.CalculateDeploymentHealthStatus(deployment.Status)
			workloadManifests[model.K8sWorkloadID{
				Namespace: deployment.Namespace,
				Kind:      model.K8sResourceKindDeployment,
				Name:      deployment.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    deployment.Status.AvailableReplicas,
				Selector:             deployment.Spec.Selector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindDaemonSet:
			daemonset := &appsv1.DaemonSet{}
			err := k8sCacheClient.Get(ctx, client.ObjectKey{
				Namespace: filters.NamespaceString,
				Name:      filters.SingleWorkload.WorkloadName,
			}, daemonset)
			if err != nil {
				return nil, client.IgnoreNotFound(err)
			}
			workloadHealthStatus := status.CalculateDaemonSetHealthStatus(daemonset.Status)
			workloadManifests[model.K8sWorkloadID{
				Namespace: daemonset.Namespace,
				Kind:      model.K8sResourceKindDaemonSet,
				Name:      daemonset.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    daemonset.Status.NumberReady,
				Selector:             daemonset.Spec.Selector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindStatefulSet:
			statefulset := &appsv1.StatefulSet{}
			err := k8sCacheClient.Get(ctx, client.ObjectKey{
				Namespace: filters.NamespaceString,
				Name:      filters.SingleWorkload.WorkloadName,
			}, statefulset)
			if err != nil {
				return nil, client.IgnoreNotFound(err)
			}
			workloadHealthStatus := status.CalculateStatefulSetHealthStatus(statefulset.Status)
			workloadManifests[model.K8sWorkloadID{
				Namespace: statefulset.Namespace,
				Kind:      model.K8sResourceKindStatefulSet,
				Name:      statefulset.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    statefulset.Status.ReadyReplicas,
				Selector:             statefulset.Spec.Selector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindCronJob:
			cronjob := &batchv1.CronJob{}
			err := k8sCacheClient.Get(ctx, client.ObjectKey{
				Namespace: filters.NamespaceString,
				Name:      filters.SingleWorkload.WorkloadName,
			}, cronjob)
			if err != nil {
				return nil, client.IgnoreNotFound(err)
			}
			workloadHealthStatus := status.CalculateCronJobHealthStatus(cronjob.Status)
			workloadManifests[model.K8sWorkloadID{
				Namespace: cronjob.Namespace,
				Kind:      model.K8sResourceKindCronJob,
				Name:      cronjob.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    int32(len(cronjob.Status.Active)),
				Selector:             cronjob.Spec.JobTemplate.Spec.Selector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindDeploymentConfig:
			// Only try to get DeploymentConfig if it's available in the cluster
			if !kube.IsOpenShiftDeploymentConfigAvailable {
				return nil, nil
			}

			// Use dynamic client for DeploymentConfig
			gvr := schema.GroupVersionResource{
				Group:    "apps.openshift.io",
				Version:  "v1",
				Resource: "deploymentconfigs",
			}

			unstructuredDC, err := timedAPICall(
				logger,
				fmt.Sprintf("Get DeploymentConfig %s/%s", filters.NamespaceString, filters.SingleWorkload.WorkloadName),
				func() (*openshiftappsv1.DeploymentConfig, error) {
					uDC, err := kube.DefaultClient.DynamicClient.Resource(gvr).Namespace(filters.NamespaceString).Get(ctx, filters.SingleWorkload.WorkloadName, metav1.GetOptions{})
					if err != nil {
						return nil, err
					}
					var dc openshiftappsv1.DeploymentConfig
					err = runtime.DefaultUnstructuredConverter.FromUnstructured(uDC.Object, &dc)
					return &dc, err
				},
			)
			if err != nil {
				if apierrors.IsNotFound(err) {
					// workload can be not found and it is not an error.
					// we will just skip it.
					return nil, nil
				}
				return nil, err
			}
			workloadHealthStatus := status.CalculateDeploymentConfigHealthStatus(unstructuredDC.Status)

			// Convert map[string]string selector to *metav1.LabelSelector
			labelSelector := &metav1.LabelSelector{
				MatchLabels: unstructuredDC.Spec.Selector,
			}

			workloadManifests[model.K8sWorkloadID{
				Namespace: unstructuredDC.Namespace,
				Kind:      model.K8sResourceKindDeploymentConfig,
				Name:      unstructuredDC.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    unstructuredDC.Status.AvailableReplicas,
				Selector:             labelSelector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindArgoRollout:
			if !kube.IsArgoRolloutAvailable {
				return nil, nil
			}

			rollout := &argorolloutsv1alpha1.Rollout{}
			err := k8sCacheClient.Get(ctx, client.ObjectKey{
				Namespace: filters.NamespaceString,
				Name:      filters.SingleWorkload.WorkloadName,
			}, rollout)
			if err != nil {
				return nil, client.IgnoreNotFound(err)
			}
			workloadHealthStatus := status.CalculateRolloutHealthStatus(rollout.Status)
			workloadManifests[model.K8sWorkloadID{
				Namespace: rollout.Namespace,
				Kind:      model.K8sResourceKindRollout,
				Name:      rollout.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    rollout.Status.AvailableReplicas,
				Selector:             rollout.Spec.Selector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
			return workloadManifests, nil

		default:
			return nil, fmt.Errorf("invalid workload kind: %s", filters.SingleWorkload.WorkloadKind)
		}
	}

	deploymentsList := &appsv1.DeploymentList{}
	err = k8sCacheClient.List(ctx, deploymentsList, client.InNamespace(filters.NamespaceString), client.MatchingLabels(map[string]string{}))
	if err != nil {
		return nil, err
	}
	deploymentsMap := make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
	for _, deployment := range deploymentsList.Items {
		workloadHealthStatus := status.CalculateDeploymentHealthStatus(deployment.Status)
		deploymentsMap[model.K8sWorkloadID{
			Namespace: deployment.Namespace,
			Kind:      model.K8sResourceKindDeployment,
			Name:      deployment.Name,
		}] = &computed.CachedWorkloadManifest{
			AvailableReplicas:    deployment.Status.AvailableReplicas,
			Selector:             deployment.Spec.Selector,
			WorkloadHealthStatus: workloadHealthStatus,
		}
	}

	daemonsetsList := &appsv1.DaemonSetList{}
	err = k8sCacheClient.List(ctx, daemonsetsList, client.InNamespace(filters.NamespaceString), client.MatchingLabels(map[string]string{}))
	if err != nil {
		return nil, err
	}
	daemonsMap := make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
	for _, daemonset := range daemonsetsList.Items {
		workloadHealthStatus := status.CalculateDaemonSetHealthStatus(daemonset.Status)
		daemonsMap[model.K8sWorkloadID{
			Namespace: daemonset.Namespace,
			Kind:      model.K8sResourceKindDaemonSet,
			Name:      daemonset.Name,
		}] = &computed.CachedWorkloadManifest{
			AvailableReplicas:    daemonset.Status.NumberReady,
			Selector:             daemonset.Spec.Selector,
			WorkloadHealthStatus: workloadHealthStatus,
		}
	}

	statefulsetsList := &appsv1.StatefulSetList{}
	err = k8sCacheClient.List(ctx, statefulsetsList, client.InNamespace(filters.NamespaceString), client.MatchingLabels(map[string]string{}))
	if err != nil {
		return nil, err
	}
	statefulsetsMap := make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
	for _, statefulset := range statefulsetsList.Items {
		workloadHealthStatus := status.CalculateStatefulSetHealthStatus(statefulset.Status)
		statefulsetsMap[model.K8sWorkloadID{
			Namespace: statefulset.Namespace,
			Kind:      model.K8sResourceKindStatefulSet,
			Name:      statefulset.Name,
		}] = &computed.CachedWorkloadManifest{
			AvailableReplicas:    statefulset.Status.ReadyReplicas,
			Selector:             statefulset.Spec.Selector,
			WorkloadHealthStatus: workloadHealthStatus,
		}
	}

	cronjobsList := &batchv1.CronJobList{}
	err = k8sCacheClient.List(ctx, cronjobsList, client.InNamespace(filters.NamespaceString), client.MatchingLabels(map[string]string{}))
	if err != nil {
		return nil, err
	}
	cronjobsMap := make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
	for _, cronjob := range cronjobsList.Items {
		workloadHealthStatus := status.CalculateCronJobHealthStatus(cronjob.Status)
		cronjobsMap[model.K8sWorkloadID{
			Namespace: cronjob.Namespace,
			Kind:      model.K8sResourceKindCronJob,
			Name:      cronjob.Name,
		}] = &computed.CachedWorkloadManifest{
			AvailableReplicas:    int32(len(cronjob.Status.Active)),
			Selector:             cronjob.Spec.JobTemplate.Spec.Selector,
			WorkloadHealthStatus: workloadHealthStatus,
		}
	}

	// Only try to list DeploymentConfigs if they're available in the cluster
	// This avoids permission errors on non-OpenShift clusters
	deploymentconfigsMap := make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
	if kube.IsDeploymentConfigAvailable() {

		// Use dynamic client for DeploymentConfigs
		gvr := schema.GroupVersionResource{
			Group:    "apps.openshift.io",
			Version:  "v1",
			Resource: "deploymentconfigs",
		}

		dcListUnstructured, err := timedAPICall(
			logger,
			formatOperationMessage("List DeploymentConfigs", filters.NamespaceString),
			func() ([]openshiftappsv1.DeploymentConfig, error) {
				uList, err := kube.DefaultClient.DynamicClient.Resource(gvr).Namespace(filters.NamespaceString).List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}

				dcList := make([]openshiftappsv1.DeploymentConfig, 0, len(uList.Items))
				for _, uDC := range uList.Items {
					var dc openshiftappsv1.DeploymentConfig
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uDC.Object, &dc); err != nil {
						// Log the error but continue with other items
						logger.Error(err, "failed to convert DeploymentConfig", "name", uDC.GetName())
						continue
					}
					dcList = append(dcList, dc)
				}
				return dcList, nil
			},
		)
		if err != nil {
			return nil, err
		}

		for _, dc := range dcListUnstructured {
			workloadHealthStatus := status.CalculateDeploymentConfigHealthStatus(dc.Status)

			// Convert map[string]string selector to *metav1.LabelSelector
			labelSelector := &metav1.LabelSelector{
				MatchLabels: dc.Spec.Selector,
			}

			deploymentconfigsMap[model.K8sWorkloadID{
				Namespace: dc.Namespace,
				Kind:      model.K8sResourceKindDeploymentConfig,
				Name:      dc.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    dc.Status.AvailableReplicas,
				Selector:             labelSelector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
		}
	}

	rolloutsMap := make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
	if kube.IsArgoRolloutAvailable {
		rolloutsList := &argorolloutsv1alpha1.RolloutList{}
		err := k8sCacheClient.List(ctx, rolloutsList, client.InNamespace(filters.NamespaceString), client.MatchingLabels(map[string]string{}))
		if err != nil {
			return nil, err
		}

		for _, rollout := range rolloutsList.Items {
			workloadHealthStatus := status.CalculateRolloutHealthStatus(rollout.Status)
			rolloutsMap[model.K8sWorkloadID{
				Namespace: rollout.Namespace,
				Kind:      model.K8sResourceKindRollout,
				Name:      rollout.Name,
			}] = &computed.CachedWorkloadManifest{
				AvailableReplicas:    rollout.Status.AvailableReplicas,
				Selector:             rollout.Spec.Selector,
				WorkloadHealthStatus: workloadHealthStatus,
			}
		}
	}

	workloadManifests = make(map[model.K8sWorkloadID]*computed.CachedWorkloadManifest)
	for id, manifest := range deploymentsMap {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range statefulsetsMap {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range daemonsMap {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range cronjobsMap {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range deploymentconfigsMap {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range rolloutsMap {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}

	return workloadManifests, nil
}

func fetchWorkloadPods(ctx context.Context, logger logr.Logger, filters *WorkloadFilter, singleWorkloadManifest *computed.CachedWorkloadManifest, workloadIdsMap map[k8sconsts.PodWorkload]struct{}, k8sCacheClient client.Client) (workloadPods map[model.K8sWorkloadID][]*corev1.Pod, err error) {

	var labelSelector *metav1.LabelSelector
	if filters.SingleWorkload != nil {
		if singleWorkloadManifest == nil || singleWorkloadManifest.Selector == nil {
			// if workload is not found for this pod, skip the queries - no pods to fetch.
			return map[model.K8sWorkloadID][]*corev1.Pod{}, nil
		}
		labelSelector = singleWorkloadManifest.Selector
	}

	podList := &corev1.PodList{}
	opts := []client.ListOption{client.InNamespace(filters.NamespaceString)}
	if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid label selector: %w", err)
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}
	err = k8sCacheClient.List(ctx, podList, opts...)
	if err != nil {
		return nil, err
	}

	workloadPods = make(map[model.K8sWorkloadID][]*corev1.Pod)
	for _, pod := range podList.Items {
		if _, ok := filters.IgnoredNamespaces[pod.Namespace]; ok {
			continue
		}
		pw, err := workload.PodWorkloadObject(ctx, &pod)
		if err != nil || pw == nil {
			// skip pods not relevant for odigos
			continue
		}
		if _, ok := workloadIdsMap[*pw]; !ok {
			// fmt.Printf("skipping pod %s/%s because it is not relevant for odigos\n", pod.Namespace, pod.Name)
			// skip pods not relevant for odigos.
			// for example, when we are fetching only instrumentated workloads,
			// we can drop all the pods which does not participate.
			continue
		}

		workloadId := model.K8sWorkloadID{
			Namespace: pod.Namespace,
			Kind:      model.K8sResourceKind(pw.Kind),
			Name:      pw.Name,
		}
		workloadPods[workloadId] = append(workloadPods[workloadId], &pod)
	}
	return workloadPods, nil
}

func fetchInstrumentationInstances(ctx context.Context, logger logr.Logger, filters *WorkloadFilter, k8sCacheClient client.Client) (
	byPodContainer map[PodContainerId][]*odigosv1.InstrumentationInstance,
	byWorkloadContainer map[WorkloadContainerId][]*odigosv1.InstrumentationInstance,
	err error) {

	var matchingLabels map[string]string
	if filters.SingleWorkload != nil {
		// fetch only the instrumentation instances for the specific workload.
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(filters.SingleWorkload.WorkloadName, k8sconsts.WorkloadKind(filters.SingleWorkload.WorkloadKind))
		matchingLabels = map[string]string{
			consts.InstrumentedAppNameLabel: instrumentationConfigName,
		}
	}

	var ii odigosv1.InstrumentationInstanceList
	opts := []client.ListOption{client.InNamespace(filters.NamespaceString)}
	if matchingLabels != nil {
		opts = append(opts, client.MatchingLabels(matchingLabels))
	}
	err = k8sCacheClient.List(ctx, &ii, opts...)
	if err != nil {
		return nil, nil, err
	}

	byPodContainer = make(map[PodContainerId][]*odigosv1.InstrumentationInstance, len(ii.Items))
	byWorkloadContainer = make(map[WorkloadContainerId][]*odigosv1.InstrumentationInstance, len(ii.Items))
	for _, ii := range ii.Items {
		if _, ok := filters.IgnoredNamespaces[ii.Namespace]; ok {
			continue
		}
		ownerPodLabel, ok := ii.Labels[odigosv1.OwnerPodNameLabel]
		if !ok {
			// instrumentation instance must have this label
			// if it's missing for any reason, we will just skip it as we cannot use this instance.
			continue
		}

		// add to the byContainer map
		containerId := PodContainerId{
			Namespace:     ii.Namespace,
			PodName:       ownerPodLabel,
			ContainerName: ii.Spec.ContainerName,
		}
		byPodContainer[containerId] = append(byPodContainer[containerId], &ii)

		instrumentedAppLabel, ok := ii.Labels[consts.InstrumentedAppNameLabel]
		if !ok {
			continue
		}
		instrumentedAppDetails, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(instrumentedAppLabel, ii.Namespace)
		if err != nil {
			continue
		}
		workloadContainerId := WorkloadContainerId{
			Namespace:     instrumentedAppDetails.Namespace,
			Kind:          instrumentedAppDetails.Kind,
			Name:          instrumentedAppDetails.Name,
			ContainerName: ii.Spec.ContainerName,
		}
		byWorkloadContainer[workloadContainerId] = append(byWorkloadContainer[workloadContainerId], &ii)
	}
	return byPodContainer, byWorkloadContainer, nil
}
