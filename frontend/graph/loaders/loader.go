package loaders

import (
	"context"
	"sync"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type loadersKeyType string

const loadersKey loadersKeyType = "dataloaders"

type Loaders struct {
	mu sync.Mutex

	filter *model.SourceFilter

	// the value we use for the namespace in the quires to api server.
	// for all namespace, this will be empty string.
	// for a namespace query, or a query for specific source, this will be the namespace name.
	queryNamespace string

	workloadIds            []model.K8sWorkload
	instrumentationConfigs map[model.K8sWorkload]*v1alpha1.InstrumentationConfig
	workloadSources        map[model.K8sWorkload]*v1alpha1.Source
	nsSources              map[string]*v1alpha1.Source
	workloadManifests      map[model.K8sWorkload]*WorkloadManifest
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

func NewLoaders() *Loaders {
	return &Loaders{}
}

func (l *Loaders) GetWorkloadIds() []model.K8sWorkload {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.workloadIds
}

func (l *Loaders) GetInstrumentationConfig(ctx context.Context, workload model.K8sWorkload) *v1alpha1.InstrumentationConfig {
	l.mu.Lock()
	defer l.mu.Unlock()

	// if we did not fecth the instrumentation configs yet, do it now.
	if len(l.instrumentationConfigs) == 0 {
		instrumentationConfigs, err := l.fetchInstrumentationConfigs(ctx)
		if err != nil {
			return nil
		}
		l.instrumentationConfigs = instrumentationConfigs
	}
	return l.instrumentationConfigs[workload]
}

func (l *Loaders) GetSources(ctx context.Context, sourceId model.K8sWorkload) (*v1alpha1.WorkloadSources, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// if we did not fetch the sources yet, do it now.
	if len(l.workloadSources) == 0 || len(l.nsSources) == 0 {
		workloadSources, namespaceSources, err := l.fetchSources(ctx)
		if err != nil {
			return nil, err
		}
		l.workloadSources = workloadSources
		l.nsSources = namespaceSources
	}

	// return both the workload and namespace sources for this one.
	return &v1alpha1.WorkloadSources{
		Workload:  l.workloadSources[sourceId],
		Namespace: l.nsSources[sourceId.Namespace],
	}, nil
}

func (l *Loaders) SetFilters(ctx context.Context, filter *model.SourceFilter) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.filter = filter

	if filter != nil {
		if filter.Namespace != nil {
			l.queryNamespace = *filter.Namespace
		}
	}

	if filter.MarkedForInstrumentation != nil && *filter.MarkedForInstrumentation {
		configById, err := l.fetchInstrumentationConfigs(ctx)
		if err != nil {
			return err
		}
		l.instrumentationConfigs = configById
		l.workloadIds = make([]model.K8sWorkload, 0, len(configById))
		for sourceId := range configById {
			l.workloadIds = append(l.workloadIds, sourceId)
		}
	} else {
		// fetch all sources (both those marked for instrumentation and those not)
		// this is to allow the user to review and instrument potential sources.
		workloadSources, namespaceSources, err := l.fetchSources(ctx)
		if err != nil {
			return err
		}
		l.workloadSources = workloadSources
		l.nsSources = namespaceSources

		workloadManifests, err := l.fetchWorkloadManifests(ctx)
		if err != nil {
			return err
		}
		l.workloadManifests = workloadManifests

		// calculate the source ids from the workload sources and manifests.
		// we can have workloads without sources, and sources without workloads.
		allWorkloads := make(map[model.K8sWorkload]struct{})
		for workloadId := range workloadSources {
			allWorkloads[workloadId] = struct{}{}
		}
		for workloadId := range workloadManifests {
			allWorkloads[workloadId] = struct{}{}
		}
		l.workloadIds = make([]model.K8sWorkload, 0, len(allWorkloads))
		for sourceId := range allWorkloads {
			l.workloadIds = append(l.workloadIds, sourceId)
		}
	}

	return nil
}

// function to get just the instrumentation configs that match the filter.
// e.g. load only sources which are marked for instrumentation after the instrumentor reconciles it.
// this is cheaper and faster query than to load all the sources and resolve each one.
func (l *Loaders) fetchInstrumentationConfigs(ctx context.Context) (map[model.K8sWorkload]*v1alpha1.InstrumentationConfig, error) {
	instrumentationConfigs, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(l.queryNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	configById := make(map[model.K8sWorkload]*v1alpha1.InstrumentationConfig, len(instrumentationConfigs.Items))
	for _, config := range instrumentationConfigs.Items {
		pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(config.Name, config.Namespace)
		if err != nil {
			return nil, err
		}
		sourceId := model.K8sWorkload{
			Namespace: config.Namespace,
			Kind:      model.K8sResourceKind(pw.Kind),
			Name:      pw.Name,
		}
		configById[sourceId] = &config
	}
	return configById, nil
}

func (l *Loaders) fetchSources(ctx context.Context) (workloadSources map[model.K8sWorkload]*v1alpha1.Source, namespaceSources map[string]*v1alpha1.Source, err error) {
	sources, err := kube.DefaultClient.OdigosClient.Sources(l.queryNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	workloadSources = make(map[model.K8sWorkload]*v1alpha1.Source, len(sources.Items)) // assuming most source are workload so len is almost right
	namespaceSources = make(map[string]*v1alpha1.Source)                               // expecting only few of these
	for _, source := range sources.Items {
		wd := source.Spec.Workload
		sourceId := model.K8sWorkload{
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

func (l *Loaders) fetchWorkloadManifests(ctx context.Context) (workloadManifests map[model.K8sWorkload]*WorkloadManifest, err error) {

	workloadManifests = make(map[model.K8sWorkload]*WorkloadManifest)

	deployments, err := kube.DefaultClient.AppsV1().Deployments(l.queryNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, deployment := range deployments.Items {
		workloadManifests[model.K8sWorkload{
			Namespace: deployment.Namespace,
			Kind:      model.K8sResourceKindDeployment,
			Name:      deployment.Name,
		}] = &WorkloadManifest{
			AvailableReplicas: deployment.Status.AvailableReplicas,
			PodTemplateSpec:   &deployment.Spec.Template,
		}
	}

	daemonsets, err := kube.DefaultClient.AppsV1().DaemonSets(l.queryNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, daemonset := range daemonsets.Items {
		workloadManifests[model.K8sWorkload{
			Namespace: daemonset.Namespace,
			Kind:      model.K8sResourceKindDaemonSet,
			Name:      daemonset.Name,
		}] = &WorkloadManifest{
			AvailableReplicas: daemonset.Status.NumberReady,
			PodTemplateSpec:   &daemonset.Spec.Template,
		}
	}

	statefulsets, err := kube.DefaultClient.AppsV1().StatefulSets(l.queryNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, statefulset := range statefulsets.Items {
		workloadManifests[model.K8sWorkload{
			Namespace: statefulset.Namespace,
			Kind:      model.K8sResourceKindStatefulSet,
			Name:      statefulset.Name,
		}] = &WorkloadManifest{
			AvailableReplicas: statefulset.Status.ReadyReplicas,
			PodTemplateSpec:   &statefulset.Spec.Template,
		}
	}

	cronjobs, err := kube.DefaultClient.BatchV1().CronJobs(l.queryNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, cronjob := range cronjobs.Items {
		workloadManifests[model.K8sWorkload{
			Namespace: cronjob.Namespace,
			Kind:      model.K8sResourceKindCronJob,
			Name:      cronjob.Name,
		}] = &WorkloadManifest{
			AvailableReplicas: int32(len(cronjob.Status.Active)),
			PodTemplateSpec:   &cronjob.Spec.JobTemplate.Spec.Template,
		}
	}

	return
}
