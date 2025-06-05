package sourceinstrumentation

import (
	"context"
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

	enabled, markedForInstrumentationCondition, err := sourceutils.IsObjectInstrumentedBySource(ctx, k8sClient, obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	podWorkload := k8sconsts.PodWorkload{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Kind:      workload.WorkloadKindFromClientObject(obj),
	}

	if enabled {
		workloadObj, err := workload.ObjectToWorkload(obj)
		if err != nil {
			return ctrl.Result{}, err
		}

		instConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)
		ic := &v1alpha1.InstrumentationConfig{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: podWorkload.Namespace}, ic)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			ic, err = createInstrumentationConfigForWorkload(ctx, k8sClient, instConfigName, podWorkload.Namespace, obj, scheme)
			if err != nil {
				if apierrors.IsAlreadyExists(err) {
					// If we hit AlreadyExists here, we just hit a race in the api/cache and want to requeue. No need to log an error
					return ctrl.Result{Requeue: true}, nil
				}
				return ctrl.Result{}, err
			}
		}

		markedForInstChanged := meta.SetStatusCondition(&ic.Status.Conditions, markedForInstrumentationCondition)
		runtimeDetailsChanged := initiateRuntimeDetailsConditionIfMissing(ic, workloadObj)
		agentEnabledChanged := initiateAgentEnabledConditionIfMissing(ic)

		if markedForInstChanged || runtimeDetailsChanged || agentEnabledChanged {
			ic.Status.Conditions = sortIcConditionsByLogicalOrder(ic.Status.Conditions)

			err = k8sClient.Status().Update(ctx, ic)
			if err != nil {
				logger.Info("Failed to update status conditions of InstrumentationConfig", "name", instConfigName, "namespace", podWorkload.Namespace, "error", err.Error())
				return k8sutils.K8SUpdateErrorHandler(err)
			}
		}
	} else {
		return ctrl.Result{}, deleteWorkloadInstrumentationConfig(ctx, k8sClient, podWorkload)
	}

	return ctrl.Result{}, nil
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
	instConfig.Spec.ServiceName = serviceName

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
