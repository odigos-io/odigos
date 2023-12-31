package instrumentationdevice

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/common/utils"
	"github.com/keyval-dev/odigos/instrumentor/instrumentation"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func clearInstrumentationEbpf(obj client.Object) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return
	}

	delete(annotations, consts.EbpfInstrumentationAnnotation)
}

func isDataCollectionReady(ctx context.Context, c client.Client) bool {
	logger := log.FromContext(ctx)
	var collectorGroups odigosv1.CollectorsGroupList
	err := c.List(ctx, &collectorGroups, client.InNamespace(utils.GetCurrentNamespace()))
	if err != nil {
		logger.Error(err, "error getting collectors groups, skipping instrumentation")
		return false
	}

	for _, cg := range collectorGroups.Items {
		if cg.Spec.Role == odigosv1.CollectorsGroupRoleDataCollection && cg.Status.Ready {
			return true
		}
	}

	return false
}

func instrument(logger logr.Logger, ctx context.Context, kubeClient client.Client, runtimeDetails *odigosv1.InstrumentedApplication) error {
	obj, err := getTargetObject(ctx, kubeClient, runtimeDetails)
	if err != nil {
		return err
	}

	var odigosConfig odigosv1.OdigosConfiguration
	err = kubeClient.Get(ctx, client.ObjectKey{Namespace: utils.GetCurrentNamespace(), Name: "odigos-config"}, &odigosConfig)
	if err != nil {
		return err
	}

	result, err := controllerutil.CreateOrPatch(ctx, kubeClient, obj, func() error {
		podSpec, err := getPodSpecFromObject(obj)
		if err != nil {
			return err
		}

		return instrumentation.ApplyInstrumentationDevicesToPodTemplate(podSpec, runtimeDetails, odigosConfig.Spec.DefaultSDKs)
	})

	if err != nil {
		return err
	}

	if result != controllerutil.OperationResultNone {
		logger.V(0).Info("instrumented application", "name", obj.GetName(), "namespace", obj.GetNamespace())
	}

	return nil
}

func uninstrument(logger logr.Logger, ctx context.Context, kubeClient client.Client, namespace string, name string, kind string) error {
	obj, err := getObjectFromKindString(kind)
	if err != nil {
		logger.Error(err, "error getting object from kind string")
		return err
	}

	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		logger.Error(err, "error getting object")
		return err
	}

	result, err := controllerutil.CreateOrPatch(ctx, kubeClient, obj, func() error {

		// clear old ebpf instrumentation annotation, just in case it still exists
		clearInstrumentationEbpf(obj)
		podSpec, err := getPodSpecFromObject(obj)
		if err != nil {
			return err
		}

		instrumentation.Revert(podSpec)
		return nil
	})

	if err != nil {
		return err
	}

	if result != controllerutil.OperationResultNone {
		logger.V(0).Info("uninstrumented application", "name", obj.GetName(), "namespace", obj.GetNamespace())
	}

	return nil
}

func getTargetObject(ctx context.Context, kubeClient client.Client, runtimeDetails *odigosv1.InstrumentedApplication) (client.Object, error) {
	name, kind, err := utils.GetTargetFromRuntimeName(runtimeDetails.Name)
	if err != nil {
		return nil, err
	}

	obj, err := getObjectFromKindString(kind)
	if err != nil {
		return nil, err
	}

	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: runtimeDetails.Namespace,
		Name:      name,
	}, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func getPodSpecFromObject(obj client.Object) (*corev1.PodTemplateSpec, error) {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return &o.Spec.Template, nil
	case *appsv1.StatefulSet:
		return &o.Spec.Template, nil
	case *appsv1.DaemonSet:
		return &o.Spec.Template, nil
	default:
		return nil, errors.New("unknown kind")
	}
}

func getObjectFromKindString(kind string) (client.Object, error) {
	switch kind {
	case "Deployment":
		return &appsv1.Deployment{}, nil
	case "StatefulSet":
		return &appsv1.StatefulSet{}, nil
	case "DaemonSet":
		return &appsv1.DaemonSet{}, nil
	default:
		return nil, errors.New("unknown kind")
	}
}
