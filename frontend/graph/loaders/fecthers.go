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
func (l *Loaders) fetchInstrumentationConfigs(ctx context.Context) (map[model.K8sWorkloadID]*odigosv1.InstrumentationConfig, error) {

	// diffrentiate between a single source query and a namespace / cluster wide query.
	if l.isFilterSingleWorkload {
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(*l.filter.WorkloadName, k8sconsts.WorkloadKind(*l.filter.WorkloadKind))
		instrumentationConfigs, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(l.queryNamespace).Get(ctx, instrumentationConfigName, metav1.GetOptions{})
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
				Kind:      model.K8sResourceKind(*l.filter.WorkloadKind),
				Name:      *l.filter.WorkloadName,
			}: instrumentationConfigs,
		}, nil
	} else {
		instrumentationConfigs, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(l.queryNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		configById := make(map[model.K8sWorkloadID]*odigosv1.InstrumentationConfig, len(instrumentationConfigs.Items))
		for _, config := range instrumentationConfigs.Items {
			if _, ok := l.ignoredNamespacesMap[config.Namespace]; ok {
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

func (l *Loaders) fetchSourcesForWorkload(ctx context.Context, workloadName string, workloadNamespace string, workloadKind string) (*odigosv1.SourceList, error) {
	// for workload we need to fetch both the workload and namespace sources.
	selectorWorkload := metav1.LabelSelector{
		MatchLabels: map[string]string{
			k8sconsts.WorkloadNamespaceLabel: workloadNamespace,
			k8sconsts.WorkloadKindLabel:      workloadKind,
			k8sconsts.WorkloadNameLabel:      workloadName,
		},
	}
	workloadSources, err := kube.DefaultClient.OdigosClient.Sources(workloadNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&selectorWorkload),
	})
	if err != nil {
		return nil, err
	}

	selectorNamespace := metav1.LabelSelector{
		MatchLabels: map[string]string{
			k8sconsts.WorkloadNamespaceLabel: workloadNamespace,
			k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
			k8sconsts.WorkloadNameLabel:      workloadName,
		},
	}
	namespaceSources, err := kube.DefaultClient.OdigosClient.Sources(workloadNamespace).List(ctx, metav1.ListOptions{
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

func (l *Loaders) fetchSourcesForNamespace(ctx context.Context, ns string) (*odigosv1.SourceList, error) {
	// will return both "workload" sources and "namespace" sources as required
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			k8sconsts.WorkloadNamespaceLabel: ns,
		},
	}
	// assumes that sources are in the same namespace they are instrumenting (which is true at time of writing)
	return kube.DefaultClient.OdigosClient.Sources(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&selector),
	})
}

func (l *Loaders) fetchAllSources(ctx context.Context) (*odigosv1.SourceList, error) {
	sources, err := kube.DefaultClient.OdigosClient.Sources(l.queryNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	filteredSources := make([]odigosv1.Source, 0, len(sources.Items))
	for _, source := range sources.Items {
		if _, ok := l.ignoredNamespacesMap[source.Namespace]; ok {
			continue
		}
		filteredSources = append(filteredSources, source)
	}
	sources.Items = filteredSources
	return sources, nil
}

func (l *Loaders) fetchSources(ctx context.Context) (workloadSources map[model.K8sWorkloadID]*odigosv1.Source, namespaceSources map[string]*odigosv1.Source, err error) {

	var sources *odigosv1.SourceList
	if l.isFilterSingleWorkload {
		sources, err = l.fetchSourcesForWorkload(ctx, *l.filter.WorkloadName, *l.filter.Namespace, string(*l.filter.WorkloadKind))
	} else if l.isFilterSingleNamespace {
		sources, err = l.fetchSourcesForNamespace(ctx, *l.filter.Namespace)
	} else {
		sources, err = l.fetchAllSources(ctx)
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

func (l *Loaders) fetchWorkloadManifests(ctx context.Context) (workloadManifests map[model.K8sWorkloadID]*WorkloadManifest, err error) {

	// if this is a query for one specific workload, then fetch only it.
	if l.isFilterSingleWorkload {
		workloadManifests = make(map[model.K8sWorkloadID]*WorkloadManifest)
		switch *l.filter.WorkloadKind {
		case model.K8sResourceKindDeployment:
			deployment, err := kube.DefaultClient.AppsV1().Deployments(l.queryNamespace).Get(ctx, *l.filter.WorkloadName, metav1.GetOptions{})
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

		case model.K8sResourceKindDaemonSet:
			daemonset, err := kube.DefaultClient.AppsV1().DaemonSets(l.queryNamespace).Get(ctx, *l.filter.WorkloadName, metav1.GetOptions{})
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

		case model.K8sResourceKindStatefulSet:
			statefulset, err := kube.DefaultClient.AppsV1().StatefulSets(l.queryNamespace).Get(ctx, *l.filter.WorkloadName, metav1.GetOptions{})
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

		case model.K8sResourceKindCronJob:
			cronjob, err := kube.DefaultClient.BatchV1().CronJobs(l.queryNamespace).Get(ctx, *l.filter.WorkloadName, metav1.GetOptions{})
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
			return nil, fmt.Errorf("invalid workload kind: %s", *l.filter.WorkloadKind)
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
		deployments, err := kube.DefaultClient.AppsV1().Deployments(l.queryNamespace).List(ctx, metav1.ListOptions{})
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
		daemonsets, err := kube.DefaultClient.AppsV1().DaemonSets(l.queryNamespace).List(ctx, metav1.ListOptions{})
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
		statefulsets, err := kube.DefaultClient.AppsV1().StatefulSets(l.queryNamespace).List(ctx, metav1.ListOptions{})
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
		cronjobs, err := kube.DefaultClient.BatchV1().CronJobs(l.queryNamespace).List(ctx, metav1.ListOptions{})
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
		if _, ok := l.ignoredNamespacesMap[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range statefuls {
		if _, ok := l.ignoredNamespacesMap[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range daemons {
		if _, ok := l.ignoredNamespacesMap[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}
	for id, manifest := range crons {
		if _, ok := l.ignoredNamespacesMap[id.Namespace]; ok {
			continue
		}
		workloadManifests[id] = manifest
	}

	return workloadManifests, nil
}

func (l *Loaders) fetchWorkloadPods(ctx context.Context) (workloadPods map[model.K8sWorkloadID][]*corev1.Pod, err error) {

	var labelSelector string
	if l.isFilterSingleWorkload {

		l.workloadManifestsMutex.Lock()
		defer l.workloadManifestsMutex.Unlock()
		if len(l.workloadManifests) == 0 {
			workloadManifests, err := l.fetchWorkloadManifests(ctx)
			if err != nil {
				return nil, err
			}
			l.workloadManifests = workloadManifests
		}

		workloadId := model.K8sWorkloadID{
			Namespace: *l.filter.Namespace,
			Kind:      model.K8sResourceKind(*l.filter.WorkloadKind),
			Name:      *l.filter.WorkloadName,
		}

		workloadManifest, ok := l.workloadManifests[workloadId]
		if !ok || workloadManifest == nil || workloadManifest.Selector == nil {
			// if workload is not found for this pod, skip the queries - no pods to fetch.
			return map[model.K8sWorkloadID][]*corev1.Pod{}, nil
		}

		labelSelector = metav1.FormatLabelSelector(workloadManifest.Selector)
	}

	pods, err := kube.DefaultClient.CoreV1().Pods(l.queryNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	workloadPods = make(map[model.K8sWorkloadID][]*corev1.Pod)
	for _, pod := range pods.Items {
		if _, ok := l.ignoredNamespacesMap[pod.Namespace]; ok {
			continue
		}
		pw, err := workload.PodWorkloadObject(ctx, &pod)
		if err != nil || pw == nil {
			// skip pods not relevant for odigos
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

func (l *Loaders) fetchInstrumentationInstances(ctx context.Context) (instrumentationInstances map[PodId]*odigosv1.InstrumentationInstance, err error) {

	labelSelector := ""
	if l.isFilterSingleWorkload {
		// fetch only the instrumentation instances for the specific workload.
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(*l.filter.WorkloadName, k8sconsts.WorkloadKind(*l.filter.WorkloadKind))
		selector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				consts.InstrumentedAppNameLabel: instrumentationConfigName,
			},
		}
		labelSelector = metav1.FormatLabelSelector(&selector)
	}

	ii, err := kube.DefaultClient.OdigosClient.InstrumentationInstances(l.queryNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	instrumentationInstances = make(map[PodId]*odigosv1.InstrumentationInstance, len(ii.Items))
	for _, ii := range ii.Items {
		if _, ok := l.ignoredNamespacesMap[ii.Namespace]; ok {
			continue
		}
		ownerPodLabel, ok := ii.Labels[odigosv1.OwnerPodNameLabel]
		if !ok {
			// instrumentation instance must have this label
			// if it's missing for any reason, we will just skip it as we cannot use this instance.
			continue
		}
		instrumentationInstances[PodId{
			Namespace: ii.Namespace,
			PodName:   ownerPodLabel,
		}] = &ii
	}
	return instrumentationInstances, nil
}
