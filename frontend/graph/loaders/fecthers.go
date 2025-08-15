package loaders

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// function to get just the instrumentation configs that match the filter.
// e.g. load only sources which are marked for instrumentation after the instrumentor reconciles it.
// this is cheaper and faster query than to load all the sources and resolve each one.
func fetchInstrumentationConfigs(ctx context.Context, filters *WorkloadFilter) (map[model.K8sWorkloadID]*odigosv1.InstrumentationConfig, error) {

	// diffrentiate between a single source query and a namespace / cluster wide query.
	if filters.SingleWorkload != nil {
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(filters.SingleWorkload.WorkloadName, filters.SingleWorkload.WorkloadKind)
		instrumentationConfigs, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(filters.NamespaceString).Get(ctx, instrumentationConfigName, metav1.GetOptions{})
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
				Namespace: instrumentationConfigs.Namespace,
				Kind:      model.K8sResourceKind(filters.SingleWorkload.WorkloadKind),
				Name:      filters.SingleWorkload.WorkloadName,
			}: instrumentationConfigs,
		}, nil
	} else {
		instrumentationConfigs, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(filters.NamespaceString).List(ctx, metav1.ListOptions{})
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

func fetchSourcesForWorkload(ctx context.Context, filters *WorkloadFilterSingleWorkload) (*odigosv1.SourceList, error) {
	// for workload we need to fetch both the workload and namespace sources.
	selectorWorkload := metav1.LabelSelector{
		MatchLabels: map[string]string{
			k8sconsts.WorkloadNamespaceLabel: filters.Namespace,
			k8sconsts.WorkloadKindLabel:      string(filters.WorkloadKind),
			k8sconsts.WorkloadNameLabel:      filters.WorkloadName,
		},
	}
	workloadSources, err := kube.DefaultClient.OdigosClient.Sources(filters.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&selectorWorkload),
	})
	if err != nil {
		return nil, err
	}

	selectorNamespace := metav1.LabelSelector{
		MatchLabels: map[string]string{
			k8sconsts.WorkloadNamespaceLabel: filters.Namespace,
			k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
			k8sconsts.WorkloadNameLabel:      filters.Namespace,
		},
	}
	namespaceSources, err := kube.DefaultClient.OdigosClient.Sources(filters.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&selectorNamespace),
	})
	if err != nil {
		return nil, err
	}

	// merge the two lists into a odigosv1.SourceList
	sources := &odigosv1.SourceList{
		Items: append(workloadSources.Items, namespaceSources.Items...),
	}

	return sources, nil
}

func fetchSourcesForNamespace(ctx context.Context, filters *WorkloadFilterSingleNamespace) (*odigosv1.SourceList, error) {
	// will return both "workload" sources and "namespace" sources as required
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			k8sconsts.WorkloadNamespaceLabel: filters.Namespace,
		},
	}
	// assumes that sources are in the same namespace they are instrumenting (which is true at time of writing)
	return kube.DefaultClient.OdigosClient.Sources(filters.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&selector),
	})
}

func fetchAllSources(ctx context.Context, ignoredNamespaces map[string]struct{}) (*odigosv1.SourceList, error) {
	sources, err := kube.DefaultClient.OdigosClient.Sources("").List(ctx, metav1.ListOptions{})
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

func fetchSources(ctx context.Context, filters *WorkloadFilter) (workloadSources map[model.K8sWorkloadID]*odigosv1.Source, namespaceSources map[string]*odigosv1.Source, err error) {

	var sources *odigosv1.SourceList
	if filters.SingleWorkload != nil {
		sources, err = fetchSourcesForWorkload(ctx, filters.SingleWorkload)
	} else if filters.SingleNamespace != nil {
		sources, err = fetchSourcesForNamespace(ctx, filters.SingleNamespace)
	} else {
		sources, err = fetchAllSources(ctx, filters.IgnoredNamespaces)
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

func fetchWorkloadManifests(ctx context.Context, filters *WorkloadFilter) (workloadManifests map[model.K8sWorkloadID]*WorkloadManifest, err error) {

	// if this is a query for one specific workload, then fetch only it.
	if filters.SingleWorkload != nil {
		workloadManifests = make(map[model.K8sWorkloadID]*WorkloadManifest)
		switch filters.SingleWorkload.WorkloadKind {
		case k8sconsts.WorkloadKindDeployment:
			deployment, err := kube.DefaultClient.AppsV1().Deployments(filters.NamespaceString).Get(ctx, filters.SingleWorkload.WorkloadName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					// workload cam be not found and it is not an error.
					// we will just skip it.
					return nil, nil
				}
				return nil, err
			}
			workloadManifests[model.K8sWorkloadID{
				Namespace: deployment.Namespace,
				Kind:      model.K8sResourceKindDeployment,
				Name:      deployment.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: deployment.Status.AvailableReplicas,
				Selector:          deployment.Spec.Selector,
				PodTemplateSpec:   &deployment.Spec.Template,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindDaemonSet:
			daemonset, err := kube.DefaultClient.AppsV1().DaemonSets(filters.NamespaceString).Get(ctx, filters.SingleWorkload.WorkloadName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					// workload cam be not found and it is not an error.
					// we will just skip it.
					return nil, nil
				}
				return nil, err
			}
			workloadManifests[model.K8sWorkloadID{
				Namespace: daemonset.Namespace,
				Kind:      model.K8sResourceKindDaemonSet,
				Name:      daemonset.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: daemonset.Status.NumberReady,
				Selector:          daemonset.Spec.Selector,
				PodTemplateSpec:   &daemonset.Spec.Template,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindStatefulSet:
			statefulset, err := kube.DefaultClient.AppsV1().StatefulSets(filters.NamespaceString).Get(ctx, filters.SingleWorkload.WorkloadName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					// workload cam be not found and it is not an error.
					// we will just skip it.
					return nil, nil
				}
				return nil, err
			}
			workloadManifests[model.K8sWorkloadID{
				Namespace: statefulset.Namespace,
				Kind:      model.K8sResourceKindStatefulSet,
				Name:      statefulset.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: statefulset.Status.ReadyReplicas,
				Selector:          statefulset.Spec.Selector,
				PodTemplateSpec:   &statefulset.Spec.Template,
			}
			return workloadManifests, nil

		case k8sconsts.WorkloadKindCronJob:
			cronjob, err := kube.DefaultClient.BatchV1().CronJobs(filters.NamespaceString).Get(ctx, filters.SingleWorkload.WorkloadName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					// workload cam be not found and it is not an error.
					// we will just skip it.
					return nil, nil
				}
				return nil, err
			}
			workloadManifests[model.K8sWorkloadID{
				Namespace: cronjob.Namespace,
				Kind:      model.K8sResourceKindCronJob,
				Name:      cronjob.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: int32(len(cronjob.Status.Active)),
				Selector:          cronjob.Spec.JobTemplate.Spec.Selector,
				PodTemplateSpec:   &cronjob.Spec.JobTemplate.Spec.Template,
			}
			return workloadManifests, nil

		default:
			return nil, fmt.Errorf("invalid workload kind: %s", filters.SingleWorkload.WorkloadKind)
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	var (
		deps      = make(map[model.K8sWorkloadID]*WorkloadManifest)
		statefuls = make(map[model.K8sWorkloadID]*WorkloadManifest)
		daemons   = make(map[model.K8sWorkloadID]*WorkloadManifest)
		crons     = make(map[model.K8sWorkloadID]*WorkloadManifest)
	)

	g.Go(func() error {
		deployments, err := kube.DefaultClient.AppsV1().Deployments(filters.NamespaceString).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, deployment := range deployments.Items {
			deps[model.K8sWorkloadID{
				Namespace: deployment.Namespace,
				Kind:      model.K8sResourceKindDeployment,
				Name:      deployment.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: deployment.Status.AvailableReplicas,
				Selector:          deployment.Spec.Selector,
				PodTemplateSpec:   &deployment.Spec.Template,
			}
		}
		return nil
	})

	g.Go(func() error {
		daemonsets, err := kube.DefaultClient.AppsV1().DaemonSets(filters.NamespaceString).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, daemonset := range daemonsets.Items {
			daemons[model.K8sWorkloadID{
				Namespace: daemonset.Namespace,
				Kind:      model.K8sResourceKindDaemonSet,
				Name:      daemonset.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: daemonset.Status.NumberReady,
				Selector:          daemonset.Spec.Selector,
				PodTemplateSpec:   &daemonset.Spec.Template,
			}
		}
		return nil
	})

	g.Go(func() error {
		statefulsets, err := kube.DefaultClient.AppsV1().StatefulSets(filters.NamespaceString).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, statefulset := range statefulsets.Items {
			statefuls[model.K8sWorkloadID{
				Namespace: statefulset.Namespace,
				Kind:      model.K8sResourceKindStatefulSet,
				Name:      statefulset.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: statefulset.Status.ReadyReplicas,
				Selector:          statefulset.Spec.Selector,
				PodTemplateSpec:   &statefulset.Spec.Template,
			}
		}
		return nil
	})

	g.Go(func() error {
		cronjobs, err := kube.DefaultClient.BatchV1().CronJobs(filters.NamespaceString).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, cronjob := range cronjobs.Items {
			crons[model.K8sWorkloadID{
				Namespace: cronjob.Namespace,
				Kind:      model.K8sResourceKindCronJob,
				Name:      cronjob.Name,
			}] = &WorkloadManifest{
				AvailableReplicas: int32(len(cronjob.Status.Active)),
				Selector:          cronjob.Spec.JobTemplate.Spec.Selector,
				PodTemplateSpec:   &cronjob.Spec.JobTemplate.Spec.Template,
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	workloadManifests = make(map[model.K8sWorkloadID]*WorkloadManifest)
	for id, manifest := range deps {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range statefuls {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range daemons {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range crons {
		if _, ok := filters.IgnoredNamespaces[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}

	return workloadManifests, nil
}

func fetchWorkloadPods(ctx context.Context, filters *WorkloadFilter, singleWorkloadManifest *WorkloadManifest, workloadIdsMap map[k8sconsts.PodWorkload]struct{}) (workloadPods map[model.K8sWorkloadID][]*corev1.Pod, err error) {

	var labelSelector string
	if filters.SingleWorkload != nil {
		if singleWorkloadManifest == nil || singleWorkloadManifest.Selector == nil {
			// if workload is not found for this pod, skip the queries - no pods to fetch.
			return map[model.K8sWorkloadID][]*corev1.Pod{}, nil
		}
		labelSelector = metav1.FormatLabelSelector(singleWorkloadManifest.Selector)
	}

	pods, err := kube.DefaultClient.CoreV1().Pods(filters.NamespaceString).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	workloadPods = make(map[model.K8sWorkloadID][]*corev1.Pod)
	for _, pod := range pods.Items {
		if _, ok := filters.IgnoredNamespaces[pod.Namespace]; ok {
			continue
		}
		pw, err := workload.PodWorkloadObject(ctx, &pod)
		if err != nil || pw == nil {
			// skip pods not relevant for odigos
			continue
		}
		if _, ok := workloadIdsMap[*pw]; !ok {
			fmt.Printf("skipping pod %s/%s because it is not relevant for odigos\n", pod.Namespace, pod.Name)
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

func fetchInstrumentationInstances(ctx context.Context, filters *WorkloadFilter) (byContainer map[ContainerId][]*odigosv1.InstrumentationInstance, err error) {

	labelSelector := ""
	if filters.SingleWorkload != nil {
		// fetch only the instrumentation instances for the specific workload.
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(filters.SingleWorkload.WorkloadName, k8sconsts.WorkloadKind(filters.SingleWorkload.WorkloadKind))
		selector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				consts.InstrumentedAppNameLabel: instrumentationConfigName,
			},
		}
		labelSelector = metav1.FormatLabelSelector(&selector)
	}

	ii, err := kube.DefaultClient.OdigosClient.InstrumentationInstances(filters.NamespaceString).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	byContainer = make(map[ContainerId][]*odigosv1.InstrumentationInstance, len(ii.Items))
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
		containerId := ContainerId{
			Namespace:     ii.Namespace,
			PodName:       ownerPodLabel,
			ContainerName: ii.Spec.ContainerName,
		}
		byContainer[containerId] = append(byContainer[containerId], &ii)
	}
	return byContainer, nil
}
