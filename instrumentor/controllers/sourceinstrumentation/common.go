package sourceinstrumentation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	v1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func syncNamespaceWorkloads(
	ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	namespace string) (ctrl.Result, error) {

	collectiveRes := ctrl.Result{}
	var errs error
	for _, kind := range []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindStatefulSet,
	} {
		workloadObjects := workload.ClientListObjectFromWorkloadKind(kind)
		err := k8sClient.List(ctx, workloadObjects, client.InNamespace(namespace))
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		objects := make([]client.Object, 0)
		switch obj := workloadObjects.(type) {
		case *v1.DeploymentList:
			for _, dep := range obj.Items {
				objects = append(objects, &dep)
			}
		case *v1.DaemonSetList:
			for _, ds := range obj.Items {
				objects = append(objects, &ds)
			}
		case *v1.StatefulSetList:
			for _, ss := range obj.Items {
				objects = append(objects, &ss)
			}
		}

		for _, obj := range objects {
			workload := workload.ClientObjectFromWorkloadKind(kind)
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), workload)
			if client.IgnoreNotFound(err) != nil {
				return collectiveRes, err
			}
			res, err := syncWorkload(ctx, k8sClient, runtimeScheme, obj)
			if err != nil {
				errs = errors.Join(errs, err)
			}
			if !res.IsZero() {
				collectiveRes = res
			}
		}
	}
	return collectiveRes, errs
}

// syncWorkload checks if the given client.Object is instrumented by a Source.
// If not, it will attempt to delete any InstrumentationConfig for the Object.
// If it is instrumented, it will attempt to create an InstrumentationConfig if one does not exist,
// or update the existing InstrumentationConfig if necessary.
func syncWorkload(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, obj client.Object) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pw := k8sconsts.PodWorkload{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Kind:      k8sconsts.WorkloadKind(obj.GetObjectKind().GroupVersionKind().Kind),
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

	containers := make([]odigosv1.ContainerOverride, 0, len(workloadObj.PodTemplateSpec().Spec.Containers))
	for _, container := range workloadObj.PodTemplateSpec().Spec.Containers {
		// search if there is an override in the source for this container.
		// list is expected to be short (1-5 containers, so linear search is fine)
		var runtimeInfoOverride *odigosv1.RuntimeDetailsByContainer
		if sources.Workload != nil {
			for _, containerOverride := range sources.Workload.Spec.ContainerOverrides {
				if containerOverride.ContainerName == container.Name {
					runtimeInfoOverride = containerOverride.RuntimeInfo
					break
				}
			}
		}
		containers = append(containers, odigosv1.ContainerOverride{
			ContainerName: container.Name,
			RuntimeInfo:   runtimeInfoOverride,
		})
	}
	// calculate the hash for the containers overrides
	// convert to json string
	json, err := json.Marshal(containers)
	if err != nil {
		return ctrl.Result{}, err
	}
	hash := sha256.Sum256(json)
	hashString := hex.EncodeToString(hash[:16])

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	ic := &v1alpha1.InstrumentationConfig{}
	err = k8sClient.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: pw.Namespace}, ic)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		ic, err = createInstrumentationConfigForWorkload(ctx, k8sClient, instConfigName, pw.Namespace, obj, scheme, containers, hashString)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				// If we hit AlreadyExists here, we just hit a race in the api/cache and want to requeue. No need to log an error
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	} else {
		// update the instrumentation config with the new containers overrides only if it changed.
		if ic.Spec.ContainerOverridesHash != hashString {
			ic.Spec.ContainersOverrides = containers
			ic.Spec.ContainerOverridesHash = hashString
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

	dataStreamsChanged := sourceutils.HandleInstrumentationConfigDataStreamsLabels(ctx, sources, ic)

	// in case of data streams changed, we need to update the instrumentation config labels
	if dataStreamsChanged {
		err = k8sClient.Update(ctx, ic)
		if err != nil {
			logger.Info("Failed to update instrumentation config", "name", instConfigName, "namespace", pw.Namespace, "error", err.Error())
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, nil
}

func createInstrumentationConfigForWorkload(ctx context.Context, k8sClient client.Client, instConfigName string, namespace string, obj client.Object, scheme *runtime.Scheme, containers []odigosv1.ContainerOverride, containersOverridesHash string) (*v1alpha1.InstrumentationConfig, error) {
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
	instConfig.Spec.ServiceName = serviceName
	instConfig.Spec.ContainersOverrides = containers
	instConfig.Spec.ContainerOverridesHash = containersOverridesHash

	if err := ctrl.SetControllerReference(obj, &instConfig, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference", "name", instConfigName, "namespace", namespace)
		return nil, err
	}

	err = k8sClient.Create(ctx, &instConfig)
	if err != nil {
		return nil, err
	}

	logger.V(0).Info("Created instrumentation config object for workload to trigger instrumentation", "name", instConfigName, "namespace", namespace)
	return &instConfig, nil
}

func deleteWorkloadInstrumentationConfig(ctx context.Context, kubeClient client.Client, pw k8sconsts.PodWorkload) error {
	logger := log.FromContext(ctx)
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)

	err := kubeClient.Delete(ctx, &v1alpha1.InstrumentationConfig{
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
