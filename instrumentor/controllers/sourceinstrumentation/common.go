package sourceinstrumentation

import (
	"context"
	"errors"
	"slices"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type workloadKindSourceMap map[k8sconsts.WorkloadKind]map[string]*v1alpha1.Source
type reconcileFunction func(context.Context, client.Client, k8sconsts.PodWorkload, *runtime.Scheme) (ctrl.Result, error)

func syncNamespaceWorkloads(
	ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	namespace string,
	reconcileFunc reconcileFunction) (ctrl.Result, error) {

	namespaceKindSources, err := getWorkloadSourcesInNamespace(ctx, k8sClient, namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	collectiveRes := ctrl.Result{}
	var errs error
	for _, kind := range []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindStatefulSet,
	} {
		deps := workload.ClientListObjectFromWorkloadKind(kind)
		err := k8sClient.List(ctx, deps, client.InNamespace(namespace))
		if client.IgnoreNotFound(err) != nil {
			errs = errors.Join(errs, err)
			continue
		}

		objectKeys := make([]client.ObjectKey, 0)
		switch obj := deps.(type) {
		case *v1.DeploymentList:
			for _, dep := range obj.Items {
				objectKeys = append(objectKeys, client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace})
			}
		case *v1.DaemonSetList:
			for _, dep := range obj.Items {
				objectKeys = append(objectKeys, client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace})
			}
		case *v1.StatefulSetList:
			for _, dep := range obj.Items {
				objectKeys = append(objectKeys, client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace})
			}
		}

		for _, key := range objectKeys {
			// For namespace instrumentation, we only want to reconcile workloads that don't have their own explicit Source object because:
			// For instrumentation:
			//  - settings in Workload Sources take priority over settings in Namespace Sources
			//  - disabled Workload Sources prevent instrumentation
			// For uninstrumentation:
			//  - explicit Workload Sources should preserve instrumentation settings if the namespace is uninstrumented
			// TODO: The semantics of this may change for automatically created Sources (such as the UI, source grouping, non-instrumenting sources, etc)
			if _, exists := namespaceKindSources[kind][key.Name]; !exists {
				res, err := reconcileFunc(ctx, k8sClient, k8sconsts.PodWorkload{Name: key.Name, Namespace: key.Namespace, Kind: kind}, runtimeScheme)
				if err != nil {
					errs = errors.Join(errs, err)
				}
				if res.Requeue {
					collectiveRes = res
				}
			}
		}
	}
	return collectiveRes, errs
}

func uninstrumentWorkload(
	ctx context.Context,
	k8sClient client.Client,
	podWorkload k8sconsts.PodWorkload,
	scheme *runtime.Scheme) (ctrl.Result, error) {
	obj := workload.ClientObjectFromWorkloadKind(podWorkload.Kind)
	err := k8sClient.Get(ctx, client.ObjectKey{Name: podWorkload.Name, Namespace: podWorkload.Namespace}, obj)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	instrumented, _, err := sourceutils.IsObjectInstrumentedBySource(ctx, k8sClient, obj)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !instrumented {
		err = errors.Join(err, deleteWorkloadInstrumentationConfig(ctx, k8sClient, podWorkload))
		err = errors.Join(err, removeReportedNameAnnotation(ctx, k8sClient, obj))
	}
	return ctrl.Result{}, err
}

func instrumentWorkload(
	ctx context.Context,
	k8sClient client.Client,
	podWorkload k8sconsts.PodWorkload,
	scheme *runtime.Scheme) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	obj := workload.ClientObjectFromWorkloadKind(podWorkload.Kind)
	err := k8sClient.Get(ctx, client.ObjectKey{Name: podWorkload.Name, Namespace: podWorkload.Namespace}, obj)
	if err != nil {
		// Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}

	workloadObj, err := workload.ObjectToWorkload(obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	enabled, markedForInstrumentationCondition, err := sourceutils.IsObjectInstrumentedBySource(ctx, k8sClient, obj)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !enabled {
		return ctrl.Result{}, nil
	}

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)
	ic, err := createInstrumentationConfigForWorkload(ctx, k8sClient, instConfigName, podWorkload.Namespace, obj, scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	if ic == nil {
		ic = &v1alpha1.InstrumentationConfig{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: podWorkload.Namespace}, ic)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	markedForInstChanged := meta.SetStatusCondition(&ic.Status.Conditions, markedForInstrumentationCondition)
	runtimeDetailsChanged := initiateRuntimeDetailsConditionIfMissing(ic, workloadObj)
	agentEnabledChanged := initiateAgentEnabledConditionIfMissing(ic)

	if markedForInstChanged || runtimeDetailsChanged || agentEnabledChanged {
		logger.V(2).Info("Updating initial instrumentation status condition of InstrumentationConfig", "name", instConfigName, "namespace", podWorkload.Namespace)
		ic.Status.Conditions = sortIcConditionsByLogicalOrder(ic.Status.Conditions)

		err = k8sClient.Status().Update(ctx, ic)
		if err != nil {
			logger.Info("Failed to update status conditions of InstrumentationConfig", "name", instConfigName, "namespace", podWorkload.Namespace, "error", err.Error())
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, err
}

func createInstrumentationConfigForWorkload(ctx context.Context, k8sClient client.Client, instConfigName string, namespace string, obj client.Object, scheme *runtime.Scheme) (*v1alpha1.InstrumentationConfig, error) {
	logger := log.FromContext(ctx)
	instConfig := v1alpha1.InstrumentationConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "InstrumentationConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instConfigName,
			Namespace: namespace,
		},
	}

	serviceName, err := sourceutils.OtelServiceNameBySource(ctx, k8sClient, obj)
	if err != nil {
		return nil, err
	}

	if serviceName != "" {
		instConfig.Spec.ServiceName = serviceName
	}

	if err := ctrl.SetControllerReference(obj, &instConfig, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference", "name", instConfigName, "namespace", namespace)
		return nil, err
	}

	err = k8sClient.Create(ctx, &instConfig)
	if err != nil {
		return nil, client.IgnoreAlreadyExists(err)
	}

	logger.V(0).Info("Requested calculation of runtime details from odiglets", "name", instConfigName, "namespace", namespace)
	return &instConfig, nil
}

func initiateRuntimeDetailsConditionIfMissing(ic *v1alpha1.InstrumentationConfig, workloadObj workload.Workload) bool {
	if meta.FindStatusCondition(ic.Status.Conditions, v1alpha1.RuntimeDetectionStatusConditionType) != nil {
		// avoid adding the condition if it already exists
		return false
	}

	// migration code, add this condition to previous instrumentation configs
	// which were created before this condition was introduced
	if len(ic.Status.RuntimeDetailsByContainer) > 0 {
		ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.RuntimeDetectionStatusConditionType,
			Status:             metav1.ConditionTrue,
			Reason:             string(v1alpha1.RuntimeDetectionReasonWaitingForDetection),
			Message:            "runtime detection completed successfully",
			LastTransitionTime: metav1.NewTime(time.Now()),
		})
		return true
	}

	// if the workload has no available replicas, we can't detect the runtime
	if workloadObj.AvailableReplicas() == 0 {
		ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.RuntimeDetectionStatusConditionType,
			Status:             metav1.ConditionFalse,
			Reason:             string(v1alpha1.RuntimeDetectionReasonNoRunningPods),
			Message:            "No running pods available to detect source runtime",
			LastTransitionTime: metav1.NewTime(time.Now()),
		})
		return true
	}

	ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.RuntimeDetectionStatusConditionType,
		Status:             metav1.ConditionUnknown,
		Reason:             string(v1alpha1.RuntimeDetectionReasonWaitingForDetection),
		Message:            "Waiting for odiglet to initiate runtime detection in a node with running pod",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	return true
}

func initiateAgentEnabledConditionIfMissing(ic *v1alpha1.InstrumentationConfig) bool {
	if meta.FindStatusCondition(ic.Status.Conditions, v1alpha1.AgentEnabledStatusConditionType) != nil {
		// avoid adding the condition if it already exists
		return false
	}

	ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.AgentEnabledStatusConditionType,
		Status:             metav1.ConditionUnknown,
		Reason:             string(v1alpha1.AgentEnabledReasonWaitingForRuntimeInspection),
		Message:            "Waiting for runtime detection to complete",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	return true
}

// giving the input conditions array, this function will return a new array with the conditions sorted by logical order
func sortIcConditionsByLogicalOrder(conditions []metav1.Condition) []metav1.Condition {
	slices.SortFunc(conditions, func(i, j metav1.Condition) int {
		return v1alpha1.StatusConditionTypeLogicalOrder(i.Type) - v1alpha1.StatusConditionTypeLogicalOrder(j.Type)
	})
	return conditions
}

func deleteWorkloadInstrumentationConfig(ctx context.Context, kubeClient client.Client, podWorkload k8sconsts.PodWorkload) error {
	logger := log.FromContext(ctx)
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)

	err := kubeClient.Delete(ctx, &v1alpha1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: podWorkload.Namespace,
			Name:      instrumentationConfigName,
		},
	})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	logger.V(1).Info("deleted instrumentationconfig", "name", instrumentationConfigName, "namespace", podWorkload.Namespace)

	return nil
}

func removeReportedNameAnnotation(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	if _, exists := workloadObject.GetAnnotations()[consts.OdigosReportedNameAnnotation]; !exists {
		return nil
	}

	return kubeClient.Patch(ctx, workloadObject, client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"annotations":{"`+consts.OdigosReportedNameAnnotation+`":null}}}`)))
}

func getWorkloadSourcesInNamespace(ctx context.Context, k8sClient client.Client, namespace string) (workloadKindSourceMap, error) {
	// pre-process existing Sources for specific workloads so we don't have to make a bunch of API calls
	// This is used to check if a workload already has an explicit Source, so we don't overwrite its InstrumentationConfig
	sourceList := v1alpha1.SourceList{}
	// Filter out Namespace Sources, this function just checks for duplicate Workload Sources
	nonNamespaceKind, err := labels.NewRequirement(k8sconsts.WorkloadKindLabel, selection.NotIn, []string{string(k8sconsts.WorkloadKindNamespace)})
	if err != nil {
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*nonNamespaceKind)
	err = k8sClient.List(ctx, &sourceList, client.InNamespace(namespace), &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	namespaceKindSources := make(workloadKindSourceMap)
	// Initialize sub-maps for all 3 kinds
	namespaceKindSources[k8sconsts.WorkloadKindDaemonSet] = make(map[string]*v1alpha1.Source)
	namespaceKindSources[k8sconsts.WorkloadKindDeployment] = make(map[string]*v1alpha1.Source)
	namespaceKindSources[k8sconsts.WorkloadKindStatefulSet] = make(map[string]*v1alpha1.Source)
	for _, s := range sourceList.Items {
		// ex: map["Deployment"]["my-app"] = ...
		namespaceKindSources[s.Spec.Workload.Kind][s.Spec.Workload.Name] = &s
	}
	return namespaceKindSources, nil
}
