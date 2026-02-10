package sourceinstrumentation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"

	openshiftappsv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func syncNamespaceWorkloads(
	ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	namespace string,
) (ctrl.Result, error) {
	workloadsToSync := make([]k8sconsts.PodWorkload, 0)
	collectiveRes := ctrl.Result{}
	var errs error
	for _, kind := range []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindStatefulSet,
		k8sconsts.WorkloadKindCronJob,
		k8sconsts.WorkloadKindDeploymentConfig,
		k8sconsts.WorkloadKindArgoRollout,
	} {
		workloadObjects := workload.ClientListObjectFromWorkloadKind(kind)
		err := k8sClient.List(ctx, workloadObjects, client.InNamespace(namespace))
		if err != nil {
			// Ignore "no matches for kind" and "forbidden" errors for DeploymentConfig
			// This happens on non-OpenShift clusters where:
			// - The DeploymentConfig resource doesn't exist (NoMatchError)
			// - RBAC permissions aren't granted (Forbidden)
			if kind == k8sconsts.WorkloadKindDeploymentConfig && (meta.IsNoMatchError(err) || apierrors.IsForbidden(err)) {
				continue
			}
			// // Same for Argo Rollouts
			if kind == k8sconsts.WorkloadKindArgoRollout && (meta.IsNoMatchError(err) || apierrors.IsForbidden(err)) {
				continue
			}
			// For other errors or other workload kinds, collect the error
			if !meta.IsNoMatchError(err) {
				errs = errors.Join(errs, err)
			}
			continue
		}

		switch obj := workloadObjects.(type) {
		case *v1.DeploymentList:
			for _, dep := range obj.Items {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      dep.GetName(),
					Namespace: dep.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindDeployment,
				})
			}
		case *v1.DaemonSetList:
			for _, ds := range obj.Items {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      ds.GetName(),
					Namespace: ds.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindDaemonSet,
				})
			}
		case *v1.StatefulSetList:
			for _, ss := range obj.Items {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      ss.GetName(),
					Namespace: ss.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindStatefulSet,
				})
			}
		case *batchv1.CronJobList:
			for _, job := range obj.Items {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      job.GetName(),
					Namespace: job.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindCronJob,
				})
			}
		case *openshiftappsv1.DeploymentConfigList:
			for _, dc := range obj.Items {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      dc.GetName(),
					Namespace: dc.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindDeploymentConfig,
				})
			}
		case *argorolloutsv1alpha1.RolloutList:
			for _, ar := range obj.Items {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      ar.GetName(),
					Namespace: ar.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindArgoRollout,
				})
			}
		}
	}

	// add standalone pods which can be instrumented
	// pods that are owned by other higher-level object (e.g Deployment) should not accounted for here
	potentialPodWorkloads := &corev1.PodList{}
	err := k8sClient.List(ctx, potentialPodWorkloads, client.InNamespace(namespace))
	if err != nil {
		errs = errors.Join(errs, err)
	} else {
		for _, p := range potentialPodWorkloads.Items {
			if workload.IsStaticPod(&p) {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      p.Name,
					Namespace: p.Namespace,
					Kind:      k8sconsts.WorkloadKindStaticPod,
				})
			}
			// currently we only support static pods as a valid workload to instrument
			// once we add support for standalone pods, we could add a case here
		}
	}

	for _, pw := range workloadsToSync {
		res, err := syncWorkload(ctx, k8sClient, runtimeScheme, pw)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		if !res.IsZero() {
			collectiveRes = res
		}
	}

	return collectiveRes, errs
}

// syncRegexSourceWorkloads syncs all workloads that match the regex pattern in the source
func syncRegexSourceWorkloads(
	ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	source *odigosv1.Source,
) (ctrl.Result, error) {
	pattern := source.Spec.Workload.Name
	namespace := source.Spec.Workload.Namespace
	kind := source.Spec.Workload.Kind

	// Compile the regex pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Invalid regex pattern, return error
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	workloadsToSync := make([]k8sconsts.PodWorkload, 0)
	collectiveRes := ctrl.Result{}
	var errs error

	// List all workloads of the specified kind in the namespace
	workloadObjects := workload.ClientListObjectFromWorkloadKind(kind)
	err = k8sClient.List(ctx, workloadObjects, client.InNamespace(namespace))
	if err != nil {
		return ctrl.Result{}, err
	}

	// Filter workloads by regex pattern
	switch obj := workloadObjects.(type) {
	case *v1.DeploymentList:
		for _, dep := range obj.Items {
			if regex.MatchString(dep.GetName()) {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      dep.GetName(),
					Namespace: dep.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindDeployment,
				})
			}
		}
	case *v1.DaemonSetList:
		for _, ds := range obj.Items {
			if regex.MatchString(ds.GetName()) {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      ds.GetName(),
					Namespace: ds.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindDaemonSet,
				})
			}
		}
	case *v1.StatefulSetList:
		for _, ss := range obj.Items {
			if regex.MatchString(ss.GetName()) {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      ss.GetName(),
					Namespace: ss.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindStatefulSet,
				})
			}
		}
	case *batchv1.CronJobList:
		for _, job := range obj.Items {
			if regex.MatchString(job.GetName()) {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      job.GetName(),
					Namespace: job.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindCronJob,
				})
			}
		}
	case *openshiftappsv1.DeploymentConfigList:
		for _, dc := range obj.Items {
			if regex.MatchString(dc.GetName()) {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      dc.GetName(),
					Namespace: dc.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindDeploymentConfig,
				})
			}
		}
	case *argorolloutsv1alpha1.RolloutList:
		for _, rollout := range obj.Items {
			if regex.MatchString(rollout.GetName()) {
				workloadsToSync = append(workloadsToSync, k8sconsts.PodWorkload{
					Name:      rollout.GetName(),
					Namespace: rollout.GetNamespace(),
					Kind:      k8sconsts.WorkloadKindArgoRollout,
				})
			}
		}
	}

	// Sync each matching workload
	for _, pw := range workloadsToSync {
		res, err := syncWorkload(ctx, k8sClient, runtimeScheme, pw)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		if !res.IsZero() {
			collectiveRes = res
		}
	}

	return collectiveRes, errs
}

// syncWorkload checks if the given client.Object is instrumented by a Source.
// If not, it will attempt to delete any InstrumentationConfig for the Object.
// If it is instrumented, it will attempt to create an InstrumentationConfig if one does not exist,
// or update the existing InstrumentationConfig if necessary.
func syncWorkload(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, pw k8sconsts.PodWorkload) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	obj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	err := k8sClient.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, obj)
	if err != nil {
		// if err is not nil it means obj is invalid, so we must return.
		// instrumentation config has the workload as owner, so it will be deleted automatically by k8s,
		// thus NotFound is expected and we can return without error.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sources, err := odigosv1.GetSources(ctx, k8sClient, pw)
	enabled, markedForInstrumentationCondition, err := sourceutils.IsObjectInstrumentedBySource(ctx, sources, err)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !enabled {
		return ctrl.Result{}, deleteWorkloadInstrumentationConfig(ctx, k8sClient, pw)
	}

	workloadObj, err := workload.ObjectToWorkload(obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	containers := make([]odigosv1.ContainerOverride, 0, len(workloadObj.PodSpec().Containers))
	for _, container := range workloadObj.PodSpec().Containers {
		// search if there is an override in the source for this container.
		// list is expected to be short (1-5 containers, so linear search is fine)
		var containerOverride *odigosv1.ContainerOverride
		if sources.Workload != nil && !k8sutils.IsTerminating(sources.Workload) {
			for _, workloadContainerOverride := range sources.Workload.Spec.ContainerOverrides {
				if workloadContainerOverride.ContainerName == container.Name {
					containerOverride = &workloadContainerOverride
					break
				}
			}
		}

		if containerOverride != nil {
			containers = append(containers, *containerOverride)
		} else {
			// always create a container override for the container, even if it's empty.
			// this is so UI is aware of it even if runtime detection did not run or failed.
			// TODO: revisit this process in the future
			// e.g. resolve container name in gql resolver instead of persisting to instrumentation config resource.
			containers = append(containers, odigosv1.ContainerOverride{
				ContainerName: container.Name,
			})
		}
	}
	// calculate the hash for the containers overrides
	// convert to json string
	json, err := json.Marshal(containers)
	if err != nil {
		return ctrl.Result{}, err
	}
	hash := sha256.Sum256(json)
	hashString := hex.EncodeToString(hash[:16])

	desiredDataStreamsLabels := sourceutils.CalculateDataStreamsLabels(sources)
	desiredServiceName := calculateDesiredServiceName(pw, sources)

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	ic := &odigosv1.InstrumentationConfig{}
	err = k8sClient.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: pw.Namespace}, ic)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		var recoveredFromRollbackAt *metav1.Time
		if sources.Workload != nil && !k8sutils.IsTerminating(sources.Workload) {
			recoveredFromRollbackAt = sources.Workload.Spec.RecoveredFromRollbackAt
		}
		ic, err = createInstrumentationConfigForWorkload(ctx, k8sClient, instConfigName, pw.Namespace, obj, scheme, containers, hashString, desiredServiceName, desiredDataStreamsLabels, recoveredFromRollbackAt)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				// If we hit AlreadyExists here, we just hit a race in the api/cache and want to requeue. No need to log an error
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	} else {
		// update the instrumentation config with the new containers overrides only if it changed.
		dataStreamsChanged := updateDatastreamLabels(ic, desiredDataStreamsLabels)
		containerOverridesChanged := updateContainerOverride(ic, containers, hashString)
		serviceNameChanged := updateServiceName(ic, desiredServiceName)
		recoveredFromRollbackAtChanged := updateRecoveredFromRollbackAt(ic, sources)
		if containerOverridesChanged || dataStreamsChanged || serviceNameChanged || recoveredFromRollbackAtChanged {
			err = k8sClient.Update(ctx, ic)
			if err != nil {
				return k8sutils.K8SUpdateErrorHandler(err)
			}
		}
	}

	markedForInstChanged := meta.SetStatusCondition(&ic.Status.Conditions, markedForInstrumentationCondition)
	runtimeDetailsChanged := initiateRuntimeDetailsConditionIfMissing(ic, workloadObj)
	agentEnabledChanged := initiateAgentEnabledConditionIfMissing(ic)

	if markedForInstChanged || runtimeDetailsChanged || agentEnabledChanged {
		ic.Status.Conditions = sortIcConditionsByLogicalOrder(ic.Status.Conditions)

		err = k8sClient.Status().Update(ctx, ic)
		if err != nil {
			logger.Info("Failed to update status conditions of InstrumentationConfig", "name", instConfigName, "namespace", pw.Namespace, "error", err.Error())
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, nil
}

func createInstrumentationConfigForWorkload(ctx context.Context, k8sClient client.Client, instConfigName string, namespace string, obj client.Object, scheme *runtime.Scheme, containers []odigosv1.ContainerOverride, containersOverridesHash string, serviceName string, desiredDataStreamsLabels map[string]string, recoveredFromRollbackAt *metav1.Time) (*odigosv1.InstrumentationConfig, error) {
	logger := log.FromContext(ctx)
	instConfig := odigosv1.InstrumentationConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "InstrumentationConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instConfigName,
			Namespace: namespace,
			Labels:    desiredDataStreamsLabels,
		},
	}

	instConfig.Spec.ServiceName = serviceName
	instConfig.Spec.ContainersOverrides = containers
	instConfig.Spec.ContainerOverridesHash = containersOverridesHash
	instConfig.Spec.RecoveredFromRollbackAt = recoveredFromRollbackAt

	if err := ctrl.SetControllerReference(obj, &instConfig, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference", "name", instConfigName, "namespace", namespace)
		return nil, err
	}

	err := k8sClient.Create(ctx, &instConfig)
	if err != nil {
		return nil, err
	}

	logger.V(0).Info("Created instrumentation config object for workload to trigger instrumentation", "name", instConfigName, "namespace", namespace)
	return &instConfig, nil
}

func deleteWorkloadInstrumentationConfig(ctx context.Context, kubeClient client.Client, pw k8sconsts.PodWorkload) error {
	logger := log.FromContext(ctx)
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)

	err := kubeClient.Delete(ctx, &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pw.Namespace,
			Name:      instrumentationConfigName,
		},
	})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	logger.V(1).Info("deleted instrumentationconfig", "name", instrumentationConfigName, "namespace", pw.Namespace)

	return nil
}

func updateContainerOverride(ic *odigosv1.InstrumentationConfig, desiredContainers []odigosv1.ContainerOverride, desiredContainersHashString string) (updated bool) {
	if ic.Spec.ContainerOverridesHash != desiredContainersHashString {
		ic.Spec.ContainersOverrides = desiredContainers
		ic.Spec.ContainerOverridesHash = desiredContainersHashString
		return true
	}
	return false
}

func updateDatastreamLabels(instConfig *odigosv1.InstrumentationConfig, desiredLabels map[string]string) (updated bool) {
	if instConfig.Labels == nil {
		instConfig.Labels = make(map[string]string)
	}

	// Add / update labels
	for key, value := range desiredLabels {
		if instConfig.Labels[key] != value {
			instConfig.Labels[key] = value
			updated = true
		}
	}

	// Remove datastream labels not present in desiredLabels
	for key := range instConfig.Labels {
		if _, exists := desiredLabels[key]; !exists && sourceutils.IsDataStreamLabel(key) {
			delete(instConfig.Labels, key)
			updated = true
		}
	}

	return updated
}

func calculateDesiredServiceName(pw k8sconsts.PodWorkload, sources *odigosv1.WorkloadSources) string {
	// if there is no override service name, default to the workload name (deployment name etc.)
	if sources.Workload == nil ||
		k8sutils.IsTerminating(sources.Workload) ||
		sources.Workload.Spec.OtelServiceName == "" {

		return pw.Name
	}
	// otherwise, use the override service name provided by the user in source CR as is
	return sources.Workload.Spec.OtelServiceName
}

func updateServiceName(ic *odigosv1.InstrumentationConfig, desiredServiceName string) (updated bool) {
	if desiredServiceName != ic.Spec.ServiceName {
		ic.Spec.ServiceName = desiredServiceName
		return true
	}
	return false
}

func updateRecoveredFromRollbackAt(ic *odigosv1.InstrumentationConfig, sources *odigosv1.WorkloadSources) (updated bool) {
	var desired *metav1.Time
	if sources.Workload != nil && !k8sutils.IsTerminating(sources.Workload) {
		desired = sources.Workload.Spec.RecoveredFromRollbackAt
	}
	if !ic.Spec.RecoveredFromRollbackAt.Equal(desired) {
		ic.Spec.RecoveredFromRollbackAt = desired
		return true
	}
	return false
}
