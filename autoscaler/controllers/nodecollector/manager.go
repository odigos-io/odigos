package nodecollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {

	odigosNamespace := env.GetCurrentNamespace()

	err := builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-collectorsgroup").
		For(&odigosv1.CollectorsGroup{}).
		// we assume everything in the collectorsgroup spec is the configuration for the collectors to generate.
		// thus, we need to monitor any change to the spec which is what the generation field is for.
		WithEventFilter(
			predicate.Or(
				predicate.And(&odigospredicate.OdigosCollectorsGroupNodePredicate, &predicate.GenerationChangedPredicate{}),
				predicate.And(&odigospredicate.OdigosCollectorsGroupClusterPredicate),
			)).
		Complete(&CollectorsGroupReconciler{
			nodeCollectorBaseReconciler: nodeCollectorBaseReconciler{
				Client:          mgr.GetClient(),
				scheme:          mgr.GetScheme(),
				odigosNamespace: odigosNamespace,
			},
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-autosacler-deployment").
		For(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}). // in case the configmap of the node-collector is deleted or modified for any reason, this will reconcile it
		// this controller is supposed to react to:
		// 1. 	creation of the autoscaler deployment -
		// 	 	single event in the lifecycle of this pod which is part of the autoscaler deployment (one event on start up).
		//      if no sources or destinations are defined this will create a no-op config map upon a fresh install.
		// 2. 	changes to the node collector config map - which is owned by the autoscaler deployment -
		// 		if for any reason the config map is modified or deleted, this will reconcile it.
		WithEventFilter(
			predicate.Or(
				predicate.And(&odigospredicate.ObjectNamePredicate{AllowedObjectName: env.GetComponentDeploymentNameOrDefault(k8sconsts.AutoScalerDeploymentName)}, &odigospredicate.CreationPredicate{}),
				predicate.And(&odigospredicate.ObjectNamePredicate{AllowedObjectName: k8sconsts.OdigosNodeCollectorConfigMapName}, &odigospredicate.OnlyUpdatesPredicate{}),
			)).
		Complete(&AutoscalerDeploymentReconciler{
			nodeCollectorBaseReconciler: nodeCollectorBaseReconciler{
				Client:          mgr.GetClient(),
				scheme:          mgr.GetScheme(),
				odigosNamespace: odigosNamespace,
			},
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		// this controller only cares about the instrumented application existence.
		// when it is created or removed, the node collector config map needs to be updated to scrape logs for it's pods.
		WithEventFilter(&odigospredicate.ExistencePredicate{}).
		Complete(&InstrumentationConfigReconciler{
			nodeCollectorBaseReconciler: nodeCollectorBaseReconciler{
				Client:          mgr.GetClient(),
				scheme:          mgr.GetScheme(),
				odigosNamespace: odigosNamespace,
			},
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-processor").
		For(&odigosv1.Processor{}).
		// auto scaler only cares about the spec of each processor.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(&ProcessorReconciler{
			nodeCollectorBaseReconciler: nodeCollectorBaseReconciler{
				Client:          mgr.GetClient(),
				scheme:          mgr.GetScheme(),
				odigosNamespace: odigosNamespace,
			},
		})
	if err != nil {
		return err
	}

	return nil
}
